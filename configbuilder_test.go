package config

import (
	"fmt"
	"github.com/CodeNamor/http/apiclient"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"strings"
	"testing"
)

func TestDefaultConfigBuilder_Load(t *testing.T) {
	testcases := []struct {
		name          string
		path          string
		expectFileNil bool
		expectedError string
	}{
		{
			name:          "should log error, return nil and error if file not found",
			path:          "not_a_file",
			expectFileNil: true,
			expectedError: "Error opening config file",
		},
		{
			name:          "should return error and non-nil file pointer if config file present",
			path:          "testdata/example_config.json",
			expectFileNil: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			mockBuilder := defaultConfigBuilder{}
			fileResult, errResult := mockBuilder.Load(tc.path)

			if fileResult != nil {
				defer fileResult.Close()
			}

			require.Equal(t, tc.expectFileNil, fileResult == nil)

			if tc.expectedError != "" {
				require.Error(t, errResult)
				require.Contains(t, errResult.Error(), tc.expectedError)
			} else {
				require.NoError(t, errResult)
			}

		})
	}
}

func TestDefaultConfigBuilder_Read(t *testing.T) {
	testConfigFilePath := "testdata/example_config.json"
	testConfigFileData, err := os.Open(testConfigFilePath)
	require.NoError(t, err, fmt.Sprintf("Test config file is missing from path %s", testConfigFilePath))
	defer testConfigFileData.Close()

	testcases := []struct {
		name           string
		mockConfigData io.Reader
		expected       *Config
		expectedError  string
	}{
		{
			name:           "should log error, return nil and error if cannot read config data",
			mockConfigData: strings.NewReader("invalid json data"),
			expected:       nil,
			expectedError:  "Error decoding config data",
		},
		{
			name:           "should return nil and error if invalid field names in json",
			mockConfigData: strings.NewReader(`{"OopsBadField": 123}`),
			expected:       nil,
			expectedError:  "Error decoding config data json: unknown field",
		},
		{
			name:           "should write config data to config object",
			mockConfigData: testConfigFileData,
			expected: &Config{
				Hash: "b205d0e616926c8ede91e6c54377b857",
				Env:  "UnitTest",
				Port: 8000,
				Logging: LoggingConfig{
					Level: "trace",
				},
				DefaultComponentConfigs: ComponentConfigs{
					ServiceLogging: ServiceLoggingConfig{
						LogCallDuration: True,
					},
					Client: ClientConfig{
						Timeout:             10,
						IdleConnTimeout:     30,
						MaxIdleConnsPerHost: 16,
						MaxConnsPerHost:     32,
						MaxRetries:          2,
						DisableCompression:  False,
						InsecureSkipVerify:  UnSet,
						CABundlePath:        "example_cabundle.pem",
					},
				},
				ServiceConfigs: ServicesMap{
					"ABS": &ServiceConfig{
						Name:         "ABS",
						URL:          "https://some.url.com",
						AuthRequired: true,
						AuthCredentials: AuthCredentials{
							KeyComponent1: "keyc_1",
							KeyComponent2: "keyc_2",
							Euuid:         "abs_euuid",
						},
						AuthKey: "",
						EndPoints: EndpointMap{
							"ClaimStatus": &EndpointConfig{
								Name: "ClaimStatus",
								Path: "/mvClaimStatuses?",
							},
						},
						ComponentConfigOverrides: ComponentConfigs{
							ServiceLogging: ServiceLoggingConfig{
								LogCallDuration: 1,
							},
							Client: ClientConfig{
								Timeout: 30,
							},
						},
						mergedComponentConfigs: ComponentConfigs{
							ServiceLogging: ServiceLoggingConfig{
								LogCallDuration: 1,
							},
							Client: ClientConfig{
								Timeout:             30,
								IdleConnTimeout:     30,
								MaxIdleConnsPerHost: 16,
								MaxConnsPerHost:     32,
								MaxRetries:          2,
								DisableCompression:  False,
								// InsecureSkipVerify should not appear in the example config
								InsecureSkipVerify: UnSet,
								CABundlePath:       "example_cabundle.pem",
							},
						},
					},
				},
				DatabaseConfigs: DatabasesMap{
					"MDBAuth": &DatabaseConfig{
						Name:                    "MDBAuth",
						Database:                "MyDatabaseAuth",
						Server:                  "MyServer:1433",
						Username:                "MyUser",
						AuthRequired:            true,
						AuthEnvironmentVariable: "CRM_DB_PW",
					},
					"MDBNoAuth": &DatabaseConfig{
						Name:                    "MDBNoAuth",
						Database:                "MyDatabaseNoAuth",
						Server:                  "MyServer:1433",
						Username:                "MyUser",
						AuthRequired:            false,
						AuthEnvironmentVariable: "CRM_DB_PW",
					},
				},
				Options: map[string]interface{}{
					"DummyBool": true,
					// Not certain why this is decoded as float rather than int
					"DummyNum":    float64(8),
					"DummyString": "a dumb string",
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			mockBuilder := defaultConfigBuilder{}
			errResult := mockBuilder.Read(tc.mockConfigData)

			if tc.expectedError != "" {
				require.Error(t, errResult)
				require.Contains(t, errResult.Error(), tc.expectedError)
			} else {
				require.NoError(t, errResult)
			}

			require.Equal(t, tc.expected, mockBuilder.config)
		})
	}
}

func TestDefaultConfigBuilder_LoadCertPool(t *testing.T) {
	testcases := []struct {
		name           string
		mockBundlePath string
		expectedError  string
	}{
		{
			name:           "should log and return an error if the cert file does not exist",
			mockBundlePath: "not_a_bundle_file",
			expectedError:  "Error reading cert file",
		},
		{
			name: "should log and return an error if certs cannot be appended",
			// This is intentionally bad cert data
			mockBundlePath: "testdata/example_config.json",
			expectedError:  "error appending certs from cert file",
		},
		{
			name:           "should return certPool if readable and valid",
			mockBundlePath: "testdata/example_cabundle.pem",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			result, errResult := LoadCertPool(tc.mockBundlePath)

			if tc.expectedError != "" {
				require.Error(t, errResult)
				require.Nil(t, result)
				require.Contains(t, errResult.Error(), tc.expectedError)
			} else {
				require.NoError(t, errResult)
				require.NotNil(t, result)
			}

		})
	}
}

func TestDefaultConfigBuilder_LoadServiceAuthKeys(t *testing.T) {
	testcases := []struct {
		name            string
		mockServicesMap ServicesMap
		mockClient      apiclient.RetryClient
		expectedErrors  []string
		expected        ServicesMap
	}{
		{
			name: "should request no keys if there are no service configs",
		},
		{
			name: "should not request a key for services that do not require auth",
			mockServicesMap: ServicesMap{
				"A_TEST_SERVICE": &ServiceConfig{
					Name:         "A_TEST_SERVICE",
					AuthRequired: false,
				},
			},
			expected: ServicesMap{
				"A_TEST_SERVICE": &ServiceConfig{
					Name: "A_TEST_SERVICE",
				},
			},
		},
		{
			name: "should request key for service config that requires auth",
			mockServicesMap: ServicesMap{
				"A_TEST_SERVICE": &ServiceConfig{
					Name:         "A_TEST_SERVICE",
					AuthRequired: true,
				},
			},
			expected: ServicesMap{
				"A_TEST_SERVICE": &ServiceConfig{
					Name:         "A_TEST_SERVICE",
					AuthRequired: true,
					AuthKey:      "123",
				},
			},
		},
		{
			name: "should return error for services that fail when requesting key",
			mockServicesMap: ServicesMap{
				"A_TEST_SERVICE": &ServiceConfig{
					Name:         "A_TEST_SERVICE",
					AuthRequired: true,
				},
			},
			expected: ServicesMap{
				"A_TEST_SERVICE": &ServiceConfig{
					Name:         "A_TEST_SERVICE",
					AuthRequired: true,
				},
			},
			expectedErrors: []string{
				"Error retrieving auth key for A_TEST_SERVICE: This is a mock error",
			},
		},
		{
			name: "should return error for service that has empty auth key",
			mockServicesMap: ServicesMap{
				"A_TEST_SERVICE": &ServiceConfig{
					Name:         "A_TEST_SERVICE",
					AuthRequired: true,
				},
			},
			expected: ServicesMap{
				"A_TEST_SERVICE": &ServiceConfig{
					Name:         "A_TEST_SERVICE",
					AuthRequired: true,
				},
			},
			expectedErrors: []string{
				"Empty auth key for A_TEST_SERVICE",
			},
		},
		{
			name: "should request keys for all service configs and return all errors",
			mockServicesMap: ServicesMap{
				"TEST_SERVICE_1": &ServiceConfig{
					Name:         "TEST_SERVICE_1",
					AuthRequired: true,
				},
				"TEST_SERVICE_2": &ServiceConfig{
					Name:         "TEST_SERVICE_2",
					AuthRequired: true,
				},
				"TEST_SERVICE_3": &ServiceConfig{
					Name:         "TEST_SERVICE_3",
					AuthRequired: true,
				},
				"TEST_SERVICE_4": &ServiceConfig{
					Name:         "TEST_SERVICE_4",
					AuthRequired: true,
				},
			},
			expected: ServicesMap{
				"TEST_SERVICE_1": &ServiceConfig{
					Name:         "TEST_SERVICE_1",
					AuthRequired: true,
					AuthKey:      "123",
				},
				"TEST_SERVICE_2": &ServiceConfig{
					Name:         "TEST_SERVICE_2",
					AuthRequired: true,
				},
				"TEST_SERVICE_3": &ServiceConfig{
					Name:         "TEST_SERVICE_3",
					AuthRequired: true,
				},
				"TEST_SERVICE_4": &ServiceConfig{
					Name:         "TEST_SERVICE_4",
					AuthRequired: true,
					AuthKey:      "456",
				},
			},
			expectedErrors: []string{
				"Error retrieving auth key for TEST_SERVICE_2: This is a mock error",
				"Empty auth key for TEST_SERVICE_3",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			mockBuilder := defaultConfigBuilder{
				config: &Config{
					ServiceConfigs: tc.mockServicesMap,
				},
			}

			for serviceName, expectedConfig := range tc.expected {
				resultConfig, _ := mockBuilder.GetConfig().GetServiceConfig(serviceName)
				require.Equal(t, *expectedConfig, *resultConfig)
			}
		})
	}
}

func Test_mergeComponentConfigsForAllServices(t *testing.T) {
	testcases := []struct {
		name                 string
		config               Config
		expectedServiceCCABS ComponentConfigs
		expectedServiceCCDEF ComponentConfigs
		expectErr            string
	}{
		{
			name: "mixed",
			config: Config{
				DefaultComponentConfigs: ComponentConfigs{
					ServiceLogging: ServiceLoggingConfig{
						LogCallDuration: 2,
					},
					Client: ClientConfig{
						Timeout:             10,
						IdleConnTimeout:     30,
						MaxIdleConnsPerHost: 16,
						MaxConnsPerHost:     32,
						MaxRetries:          2,
						DisableCompression:  False,
						// InsecureSkipVerify should not appear in the example config
						InsecureSkipVerify: UnSet,
						CABundlePath:       "example_cabundle.pem",
					},
				},
			},
			expectedServiceCCABS: ComponentConfigs{
				ServiceLogging: ServiceLoggingConfig{
					LogCallDuration: 1,
				},
				Client: ClientConfig{
					Timeout:             99,
					IdleConnTimeout:     30,
					MaxIdleConnsPerHost: 16,
					MaxConnsPerHost:     32,
					MaxRetries:          2,
					DisableCompression:  True,
					// InsecureSkipVerify should not appear in the example config
					InsecureSkipVerify: UnSet,
					CABundlePath:       "special-abs_cabundle.pem",
				},
			},
			expectedServiceCCDEF: ComponentConfigs{
				ServiceLogging: ServiceLoggingConfig{
					LogCallDuration: 2,
				},
				Client: ClientConfig{
					Timeout:             10,
					IdleConnTimeout:     30,
					MaxIdleConnsPerHost: 1,
					MaxConnsPerHost:     2,
					MaxRetries:          3,
					DisableCompression:  False,
					// InsecureSkipVerify should not appear in the example config
					InsecureSkipVerify: UnSet,
					CABundlePath:       "example_cabundle.pem",
				},
			},
			expectErr: "",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err := mergeComponentConfigsForAllServices(&tc.config)
			if tc.expectErr == "" {
				require.NoError(t, err)
				absServiceConfig, ok := tc.config.ServiceConfigs["ABS"]
				require.True(t, ok)
				defServiceConfig, ok := tc.config.ServiceConfigs["DEF"]
				require.True(t, ok)
				require.Equal(t, tc.expectedServiceCCABS, absServiceConfig.MergedComponentConfigs())
				require.Equal(t, tc.expectedServiceCCDEF, defServiceConfig.MergedComponentConfigs())
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectErr)
			}
		})
	}
}

func Test_mergeCompConfigs(t *testing.T) {
	testcases := []struct {
		name               string
		defaultCC          *ComponentConfigs
		serviceCCO         *ComponentConfigs
		initialMergeTarget *ComponentConfigs
		expectMerged       *ComponentConfigs
		expectErr          string
	}{
		{
			name:               "all nil, returns err",
			defaultCC:          nil,
			serviceCCO:         nil,
			initialMergeTarget: nil,
			expectMerged:       nil,
			expectErr:          "nil pointer passed for mergedCC",
		},
		{
			name:      "defaultCC nil, merges serviceCC",
			defaultCC: nil,
			serviceCCO: &ComponentConfigs{
				ServiceLogging: ServiceLoggingConfig{
					LogCallDuration: 2,
				},
			},
			initialMergeTarget: &ComponentConfigs{},
			expectMerged: &ComponentConfigs{
				ServiceLogging: ServiceLoggingConfig{
					LogCallDuration: 2,
				},
			},
			expectErr: "",
		},
		{
			name: "serviceCC nil",
			defaultCC: &ComponentConfigs{
				ServiceLogging: ServiceLoggingConfig{
					LogCallDuration: 2,
				},
			},
			serviceCCO:         nil,
			initialMergeTarget: &ComponentConfigs{},
			expectMerged: &ComponentConfigs{
				ServiceLogging: ServiceLoggingConfig{
					LogCallDuration: 2,
				},
			},
			expectErr: "",
		},
		{
			name: "mixed",
			defaultCC: &ComponentConfigs{
				ServiceLogging: ServiceLoggingConfig{
					LogCallDuration: 2,
				},
				Client: ClientConfig{
					Timeout:             10,
					IdleConnTimeout:     30,
					MaxIdleConnsPerHost: 16,
					MaxConnsPerHost:     32,
					MaxRetries:          2,
					DisableCompression:  False,
					// InsecureSkipVerify should not appear in the example config
					InsecureSkipVerify: UnSet,
					CABundlePath:       "example_cabundle.pem",
				},
			},
			serviceCCO: &ComponentConfigs{
				ServiceLogging: ServiceLoggingConfig{
					LogCallDuration: 1,
				},
				Client: ClientConfig{
					Timeout:            99,
					DisableCompression: True,
				},
			},
			initialMergeTarget: &ComponentConfigs{},
			expectMerged: &ComponentConfigs{
				ServiceLogging: ServiceLoggingConfig{
					LogCallDuration: 1,
				},
				Client: ClientConfig{
					Timeout:             99,
					IdleConnTimeout:     30,
					MaxIdleConnsPerHost: 16,
					MaxConnsPerHost:     32,
					MaxRetries:          2,
					DisableCompression:  True,
					// InsecureSkipVerify should not appear in the example config
					InsecureSkipVerify: UnSet,
					CABundlePath:       "example_cabundle.pem",
				},
			},
			expectErr: "",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			mergeTarget := tc.initialMergeTarget
			err := mergeCompConfigs(tc.serviceCCO, tc.defaultCC, mergeTarget)
			if tc.expectErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectErr)
			} else {
				require.NoError(t, err)
			}
			if tc.expectMerged == nil {
				require.Nil(t, mergeTarget)
			} else {
				require.NotNil(t, mergeTarget)
				require.Equal(t, *tc.expectMerged, *mergeTarget)
			}
		})
	}
}

func Test_resolveCAPath(t *testing.T) {
	testcases := []struct {
		name     string
		jsonPath string
		certPath string
		expected string
	}{
		{
			name:     "0 - empty strings, return empty string",
			jsonPath: "",
			certPath: "",
			expected: "",
		},
		{
			name:     "1 - empty certPath, return empty string",
			jsonPath: "foo/bar.json",
			certPath: "",
			expected: "",
		},
		{
			name:     "2 - empty jsonPath, return certPath",
			jsonPath: "",
			certPath: "foo/cabundle.pem",
			expected: "foo/cabundle.pem",
		},
		{
			name:     "3 - resolves to current dir, return empty string",
			jsonPath: "foo/bar.json",
			certPath: "..",
			expected: "",
		},
		{
			name:     "4 - normal, combines cleaned path",
			jsonPath: "foo/./bar.json",
			certPath: "cabundle.pem",
			expected: "foo/cabundle.pem",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, resolveCAPath(tc.jsonPath, tc.certPath))
		})
	}
}
