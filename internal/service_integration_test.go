//go:build service_integration_test
// +build service_integration_test

package internal_test

import (
	"context"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/errgroup"
	"os"
	"simplesurance/internal"
	"testing"
)

type serviceIntegrationTestSuite struct {
	suite.Suite
}

func (suite *serviceIntegrationTestSuite) TestGetPreviousTotalRequestsWithConcurrentCalls() {
	fileName := "first_test.txt"
	f, err := os.Create(fileName)
	if err != nil {
		suite.Failf("failed test - ", "failed to create the file, %v", err)
		return
	}
	defer f.Close()
	defer os.Remove(fileName)

	requestAmount := 60
	service := internal.NewService(&internal.Config{
		FileLocation: fileName,
		PreviousTime: int64(requestAmount),
	})

	errGroup := errgroup.Group{}
	for i := 0; i < requestAmount; i++ {
		errGroup.Go(func() error {
			_, err := service.GetPreviousTotalRequests(context.Background())
			if err != nil {
				return err
			}

			return nil
		})

	}

	if err := errGroup.Wait(); err != nil {
		suite.Failf("failed test - ", "test get previous total requests with concurrent calls faield, %v", err)

		return
	}

	counter, err := service.GetPreviousTotalRequests(context.Background())

	suite.Equal(counter, requestAmount+1)
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(serviceIntegrationTestSuite))
}
