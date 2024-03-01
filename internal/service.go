package internal

import (
	"context"
	"io"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type Service struct {
	config *Config
	file   *os.File
	mu     sync.Mutex
}

type Config struct {
	FileLocation string
	PreviousTime int64
}

func NewService(config *Config) (*Service, error) {
	file, err := os.OpenFile(config.FileLocation, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "os OpenFile")
	}
	return &Service{
		config: config,
		file:   file,
	}, nil
}

func (s *Service) Shutdown() error {
	if s.file == nil {
		return nil
	}
	err := s.file.Close()
	if err != nil {
		return errors.Wrap(err, "file Close")
	}

	return nil
}

func (s *Service) GetPreviousTotalRequests(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	bytesNumber, err := s.file.WriteString(now.Format(time.RFC3339))
	if err != nil {
		return 0, errors.Wrap(err, "writeToFile")
	}

	return s.scanRows(bytesNumber, now)
}

func (s *Service) scanRows(bytesNumber int, now time.Time) (int, error) {
	var counter int
	beforeNow := now.Add(-time.Second * time.Duration(s.config.PreviousTime))
	bytesAmount := bytesNumber

	for {
		offset, err := s.file.Seek(-int64(bytesAmount), io.SeekEnd)
		if err != nil {
			return 0, errors.Wrap(err, "file Seek")
		}

		buffer := make([]byte, bytesNumber)
		_, err = s.file.Read(buffer)
		if err != nil {
			return 0, errors.Wrap(err, "file Read")
		}

		storedTime, err := time.Parse(time.RFC3339, string(buffer))
		if err != nil {
			return 0, errors.Wrap(err, "time Parse")
		}

		if storedTime.After(beforeNow) || storedTime.Equal(beforeNow) {
			counter++
			bytesAmount += bytesNumber
			if offset == 0 {
				return counter, nil
			}
			continue
		}

		break
	}

	return counter, nil
}
