package config

import "github.com/CodeNamor/http/apiclient"

type AuthKeyGetter interface {
	GetServiceKey(service *ServiceConfig, client apiclient.RetryClient) (string, error)
}
