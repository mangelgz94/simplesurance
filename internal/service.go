package internal

import (
	"context"
	"io"
	"os"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

type Service struct {
	config *Config
}

type Config struct {
	FileLocation string
	PreviousTime int64
}

func NewService(config *Config) *Service {
	return &Service{
		config: config,
	}
}

func (s *Service) GetPreviousTotalRequests(ctx context.Context) (int, error) {
	now := time.Now()
	bytesNumber, err := s.writeToFile(now)
	if err != nil {
		return 0, errors.Wrap(err, "writeToFile")
	}

	return s.scanPreviousTotalRequests(bytesNumber, now)
}

func (s *Service) writeToFile(now time.Time) (int, error) {
	file, err := os.OpenFile(s.config.FileLocation, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return 0, errors.Wrap(err, "os OpenFile")
	}
	defer file.Close()

	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
	if err != nil {
		return 0, errors.Wrap(err, "syscall Flock")
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	n, err := file.WriteString(now.Format(time.RFC3339))
	if err != nil {
		return 0, errors.Wrap(err, "file WriteString")
	}

	return n, nil
}

func (s *Service) scanPreviousTotalRequests(byteNumber int, now time.Time) (int, error) {
	file, err := os.Open(s.config.FileLocation)
	if err != nil {
		return 0, errors.Wrap(err, "os Open")
	}
	defer file.Close()

	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
	if err != nil {
		return 0, errors.Wrap(err, "syscall Flock")
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	return s.scanRows(byteNumber, file, now)
}

func (s *Service) scanRows(bytesNumber int, file *os.File, now time.Time) (int, error) {
	var counter int
	beforeNow := now.Add(-time.Second * time.Duration(s.config.PreviousTime))
	bytesAmount := bytesNumber

	for {
		offset, err := file.Seek(-int64(bytesAmount), io.SeekEnd)
		if err != nil {
			return 0, errors.Wrap(err, "file Seek")
		}

		buffer := make([]byte, bytesNumber)
		_, err = file.Read(buffer)
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
