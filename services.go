package config

import (
	"encoding/json"

	"github.com/CodeNamor/http/apiclient"
)

// ServiceConfig describes all information required to connect to a service and any of its endpoints
type ServiceConfig struct {
	Name                     string `json:"Name"`
	URL                      string `json:"Url"`
	AuthRequired             bool
	AuthEnvironmentVariable  string
	AuthCredentials          AuthCredentials
	AuthKey                  string
	EndPoints                EndpointMap
	ComponentConfigOverrides ComponentConfigs

	// mergedComponentConfigs will be populated on load as the merge
	// of DefaultComponentConfigs and ComponentConfigOverrides
	// use MergedComponentConfig() method to access the config
	mergedComponentConfigs ComponentConfigs

	HTTPClient apiclient.RetryClient `json:"-"`
}

// DatabaseConfig describes all information required to connect to a database
type DatabaseConfig struct {
	Name                    string
	Database                string
	Server                  string
	Username                string
	Password                string
	AuthRequired            bool
	AuthEnvironmentVariable string
}

// MergedComponentConfigs returns the merged component configs.
func (s *ServiceConfig) MergedComponentConfigs() ComponentConfigs {
	return s.mergedComponentConfigs
}

// ServicesMap maps the name of a service to its configuration
type ServicesMap map[string]*ServiceConfig

// DatabasesMap maps the name of a dfatabase to its configuration
type DatabasesMap map[string]*DatabaseConfig

// UnmarshalJSON reads the list of services in the config file and transforms them into a map keyed by service names
func (servicesMap *ServicesMap) UnmarshalJSON(data []byte) error {
	*servicesMap = ServicesMap{}
	var services []ServiceConfig

	if err := json.Unmarshal(data, &services); err != nil {
		return err
	}

	for _, service := range services {
		serviceCopy := service
		(*servicesMap)[service.Name] = &serviceCopy
	}

	return nil
}

// UnmarshalJSON reads the list of services in the config file and transforms them into a map keyed by service names
func (databasesMap *DatabasesMap) UnmarshalJSON(data []byte) error {
	*databasesMap = DatabasesMap{}
	var databases []DatabaseConfig

	if err := json.Unmarshal(data, &databases); err != nil {
		return err
	}

	for _, database := range databases {
		databaseCopy := database
		(*databasesMap)[database.Name] = &databaseCopy
	}

	return nil
}

// AuthCredentials describes the data necessary to request auth information
type AuthCredentials struct {
	KeyComponent1 string
	KeyComponent2 string
	Euuid         string
}

// EndpointConfig contains all information necessary to reach an endpoint of a service
type EndpointConfig struct {
	Name string
	Path string
}

// EndpointMap maps from the name of an endpoint for a service to its configuration
type EndpointMap map[string]*EndpointConfig

// UnmarshalJSON transforms a list of endpoints into a map of endpoints keyed by the endpoints' names
func (endpointMap *EndpointMap) UnmarshalJSON(data []byte) error {
	*endpointMap = EndpointMap{}
	var endpoints []EndpointConfig

	if err := json.Unmarshal(data, &endpoints); err != nil {
		return err
	}

	for _, endpoint := range endpoints {
		endpointCopy := endpoint
		(*endpointMap)[endpoint.Name] = &endpointCopy
	}

	return nil
}
