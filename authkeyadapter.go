package config

import (
	"github.com/CodeNamor/Common/errors"
	"github.com/CodeNamor/http/apiclient"
	"os"
)

type adapterService struct{}

func newAdapterService(config AuthServiceConfig) AuthKeyGetter {
	return adapterService{}
}

func (s adapterService) GetServiceKey(service *ServiceConfig, client apiclient.RetryClient) (string, error) {
	if service.AuthEnvironmentVariable != "" {
		return s.getEnvironmentKey(service.AuthEnvironmentVariable)
	}
	return "", nil
}

func (s adapterService) getEnvironmentKey(environmentVariable string) (authKey string, err error) {
	if authKey = os.Getenv(environmentVariable); authKey == "" {
		return "", errors.New("Empty auth key for '" + environmentVariable + "'")
	}
	return
}
