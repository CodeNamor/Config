package config

/*
Config is used to load configuration settings from a config.json file
located in the same folder as the main.go file.
The config.json will have this structure:
{
	"Env": "Dev",								the environment for which this config applies
    "Port": 8000, 					the port number used by this API
	"Logging: {
		"Level":"trace"
		"GrayLogURL": "10.0.0.116:12201",			gray log url and port this API logs to
	},
	"DefaultComponentConfigs": {
		"ServiceLogging": {
			"LogCallDuration":2						switch for logging call durations when this API calls other services
		},

		"Client": {                           // see https://confluence.centene.com/pages/viewpage.action?pageId=76981789
			"Timeout": 10,
			"IdleConnTimeout": 30,
			"MaxIdleConnsPerHost": 16,
			"MaxConnsPerHost": 32,
			"MaxRetries": 0,
			"DisableCompression": false,
			"InsecureSkipVerify": false,
			"CABundlePath": "caBundle.pem"    // path to certificate bundle
		},
	}
	"ServiceConfigs": [
		{
			"Name": "ABS",
			"Url": "https://some.url.com",

			"AuthRequired": true,
			"AuthCredentials": {
				"KeyComponent1": "",
				"KeyComponent2": "",
				"Euuid": ""
			},
			"AuthKey": "",

			"Endpoints": [
				"Name": "ClaimStatus",
				"Path": "/mvClaimStatuses?",
			],

			"ComponentConfigOverrides":{
				"Client": {
					// override any global client properties
				},
				"ServiceLogging": {
					// override any global logging properties
				}
			}
		},
	],

	"AuthServiceConfig": {
		"Url": "http://www.secure.org",
		"Uid": "",
		"Pwd": ""
	}

	"Options": {
		"TRMemberInquiry": true
	}
}
*/
import (
	"fmt"
	"os"
	"strings"

	"github.com/CodeNamor/HTTP/apiclient"
	"github.com/kr/pretty"
	log "github.com/sirupsen/logrus"
)

// Config models the configuration settings read from a config file. Any changes to this struct will potentially break
// other api services utilizing it for their configuration. If you feel a need to modify this, consider if your values
// can be stored in the Options property.
type Config struct {
	Env     string
	Port    int
	Logging LoggingConfig
	//DefaultComponentConfigs contains the default settings for logging and clients as related to services, these
	//can be overridden by individual service configs in their component configs
	DefaultComponentConfigs ComponentConfigs

	ServiceConfigs  ServicesMap
	DatabaseConfigs DatabasesMap
	//Options is a catch-all property for config values that are specific to an API. Use this to add custom config
	//keys before modifying this Config structure.
	Options map[string]interface{}

	// DefaultHTTPClient returns a client for http communication that uses
	// default configuration settings: there are no overrides for the
	// config values as is done for services, Services need to use their
	// own ServiceConfig.HTTPClient
	DefaultHTTPClient apiclient.RetryClient `json:"-"`

	// A unique identifier for the config file that backs this struct
	Hash string
}

// LoggingConfig holds the string representation of the logging level and the graylog URL.
type LoggingConfig struct {
	Level string
}

// ComponentConfigs describe individual categories of components that can be configured at default level and, more
// specifically, per each service
type ComponentConfigs struct {
	ServiceLogging ServiceLoggingConfig
	Client         ClientConfig
}

// ServiceLoggingConfig contains config values related to logging that can be overridden by services
type ServiceLoggingConfig struct {
	LogCallDuration configFlag `json:"LogCallDuration"`
}

// ClientConfig contains config values related to a client for communicating with a service with properties that
// can be override by services
type ClientConfig struct {
	Timeout             int
	IdleConnTimeout     int
	MaxIdleConnsPerHost int
	MaxConnsPerHost     int
	MaxRetries          int
	DisableCompression  configFlag
	InsecureSkipVerify  configFlag
	CABundlePath        string
}

// configFlag indicates a boolean value in the config file that can be of three states: False (1), True (2), or UnSet(0)
// indicating that no value was given for the value in the config. This helps us distinguish between when a boolean
// config value was set to false or whether it was not set at all, which is necessary because the zero value of a
// boolean property is false, leading to ambiguity about whether a config property was set to false or not set at all.
type configFlag int

// ConfigFlag constants
const (
	UnSet configFlag = iota
	False
	True
)

type clientFromConfigFn func(ClientConfig) apiclient.RetryClient

// New takes a config file path and name and returns a pointer to a loaded Config
func New(configPath string) (*Config, []error) {
	return newConfig(&defaultConfigBuilder{}, apiclient.NewExtendedHTTPClient, configPath)
}

func newConfig(builder configBuilder, retryClientBuilderFn RetryClientBuilderFn, configPath string) (*Config, []error) {
	var err error

	configFile, err := builder.Load(configPath)
	if err != nil {
		return nil, []error{err}
	}
	defer configFile.Close()

	err = builder.Read(configFile)
	if err != nil {
		return nil, []error{err}
	}

	buildClientFn, err := builder.InitClientFn(retryClientBuilderFn)
	if err != nil {
		return nil, []error{err}
	}

	builder.GetConfig().DefaultHTTPClient = buildClientFn(builder.GetConfig().DefaultComponentConfigs.Client)

	// merge service Overrides with defaults
	for _, serviceConfig := range builder.GetConfig().ServiceConfigs {
		// log the merged settings that will govern each ServiceConfig
		log.Info(pretty.Sprintf("ServiceName: %v ServiceConfigs.MergedComponentConfigs: %v", serviceConfig.Name, serviceConfig.MergedComponentConfigs()))
	}

	// prepare each service client
	for _, serviceConfig := range builder.GetConfig().ServiceConfigs {
		serviceConfig.HTTPClient = buildClientFn(serviceConfig.MergedComponentConfigs().Client)
	}

	return builder.GetConfig(), []error{}
}

// GetServiceConfig returns a service configuration by name
func (c *Config) GetServiceConfig(name string) (service *ServiceConfig, err error) {
	service, ok := c.ServiceConfigs[name]

	if !ok {
		err = fmt.Errorf("unable to locate service configuration for %v", name)
	}

	return
}

// GetDatabaseConfig returns a service configuration by name
func (c *Config) GetDatabaseConfig(name string) (database *DatabaseConfig, err error) {
	database, ok := c.DatabaseConfigs[name]

	if ok {
		if database.AuthRequired {
			database.Password = os.Getenv(database.AuthEnvironmentVariable)
		}
	} else {
		err = fmt.Errorf("unable to locate database configuration for %v", name)
	}

	return
}

// IsLocal returns true if the configuration is for a local machine
func (c *Config) IsLocal() bool {
	return strings.ToLower(c.Env) == "local"
}

// OptionAsString - a little syntactic sugar for fetching of entries in Options to strings
func (c *Config) OptionAsString(option string) string {
	return fmt.Sprintf("%v", c.Options[option])
}
