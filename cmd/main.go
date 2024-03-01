package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/mangelgz94/simplesurance/internal"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

var httpServer *http.Server

type service interface {
	Shutdown() error
}

var fileService service

func main() {
	logrus.Info("Starting api")
	errGroup := new(errgroup.Group)
	errGroup.Go(func() error {
		serviceConfig := &internal.Config{
			FileLocation: "../files_repository/file.txt",
			PreviousTime: 60,
		}
		apiPort := 8090

		if os.Getenv("API_PORT") != "" {
			envAPIPort, err := strconv.Atoi(os.Getenv("API_PORT"))
			if err != nil {
				logrus.Errorf("invalid api port, %s", os.Getenv("API_PORT"))
				os.Exit(0)
			}

			apiPort = envAPIPort
		}

		if os.Getenv("FILE_LOCATION") != "" {
			serviceConfig.FileLocation = os.Getenv("FILE_LOCATION")
		}

		if os.Getenv("PREVIOUS_TIME") != "" {
			previousTime, err := strconv.ParseUint(os.Getenv("PREVIOUS_TIME"), 10, 64)
			if err != nil {
				logrus.Errorf("invalid previous time, %s", os.Getenv("PREVIOUS_TIME"))
				os.Exit(0)
			}

			serviceConfig.PreviousTime = int64(previousTime)
		}

		serverConfig := &internal.ServerConfig{
			MaxConnections: 5,
			SleepTime:      2,
		}

		if os.Getenv("MAX_CONNECTIONS") != "" {
			envMaxConnections, err := strconv.Atoi(os.Getenv("MAX_CONNECTIONS"))
			if err != nil {
				logrus.Errorf("invalid max connections, %s", os.Getenv("MAX_CONNECTIONS"))
				os.Exit(0)
			}

			serverConfig.MaxConnections = envMaxConnections
		}

		if os.Getenv("SLEEP_TIME") != "" {
			envSleepTime, err := strconv.Atoi(os.Getenv("SLEEP_TIME"))
			if err != nil {
				logrus.Errorf("invalid sleep time, %s", os.Getenv("SLEEP_TIME"))
				os.Exit(0)
			}

			serverConfig.SleepTime = envSleepTime
		}

		service, err := internal.NewService(serviceConfig)
		if err != nil {
			return errors.Wrap(err, "internal NewService")
		}
		fileService = service

		httpServer = &http.Server{
			Handler: internal.NewHandler(service, serverConfig),
			Addr:    fmt.Sprintf(":%d", apiPort),
		}

		err = httpServer.ListenAndServe()
		if err != nil {

			return errors.Wrap(err, "httpServer ListenAndServe")
		}

		return nil
	})

	errGroup.Go(func() error {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
		<-c
		logrus.Info("API stopping")
		httpServer.Shutdown(context.Background())
		fileService.Shutdown()
		os.Exit(0)
		return nil
	})

	if err := errGroup.Wait(); err != nil {
		fmt.Printf("exit reason: %s \n", err)
	}
}
