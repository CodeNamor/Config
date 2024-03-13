package config

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/imdario/mergo"
	"github.com/jinzhu/copier"
	log "github.com/sirupsen/logrus"

	cnErrors "github.com/CodeNamor/Common/errors"
	compath "github.com/CodeNamor/Common/path"
	"github.com/CodeNamor/http/apiclient"
)

type configBuilder interface {
	Load(string) (*os.File, error)
	Read(io.Reader) error
	InitClientFn(RetryClientBuilderFn) (clientFromConfigFn, error)
	LoadServiceAuthKeys(AuthKeyGetter, apiclient.RetryClient) []error
	GetConfig() *Config
	GetConfigPath() string
}

type defaultConfigBuilder struct {
	config     *Config
	configPath string
}

func (b *defaultConfigBuilder) GetConfig() *Config {
	return b.config
}

func (b *defaultConfigBuilder) GetConfigPath() string {
	return b.configPath
}

// LoadCertPool reads certificates from a CA bundle file and loads them into a certificate pool
func LoadCertPool(caBundlePath string) (*x509.CertPool, error) {
	certData, err := ioutil.ReadFile(caBundlePath)
	if err != nil {
		errMsg := &cnErrors.ErrorLog{
			RootCause: "Error reading cert file " + caBundlePath,
			Err:       err,
		}
		return nil, errMsg
	}

	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM(certData)

	if !ok {
		errMsg := errors.New("error appending certs from cert file")
		return nil, errMsg
	}

	return certPool, nil
}

// RetryClientBuilderFn is the variable for holding the function that will be used to build the retry client during building of the configuration.
type RetryClientBuilderFn func(int, *http.Client) apiclient.RetryClient

type bundleMap map[string]*x509.CertPool

// loadCABundle checks bundleMap to see if a caBundle exists and uses it if found
// otherwise it loads the cleanedCABundlePath and stores it by original caBundlePath
// so it can be found later
func loadCABundle(bundleMap bundleMap, cleanedCABundlePath string, caBundlePath string) error {
	if caBundlePath == "" { // nothing to load
		return nil
	}

	_, ok := bundleMap[caBundlePath]
	if !ok { // not yet loaded
		certPool, err := LoadCertPool(cleanedCABundlePath)
		if err != nil {
			return err
		}
		bundleMap[caBundlePath] = certPool
	}
	return nil
}

// InitClientFn initializes the function used to construct a client
// for a service loading the cabundles.
// You will provide the serviceConfig.MergedComponentConfigs().Client as the value to the getClient fn
func (b *defaultConfigBuilder) InitClientFn(rbfn RetryClientBuilderFn) (clientFromConfigFn, error) {
	DefaultHTTPClientConfig := b.config.DefaultComponentConfigs.Client
	mapCertPools := make(bundleMap)
	defCleanedCAPath := resolveCAPath(b.GetConfigPath(), DefaultHTTPClientConfig.CABundlePath)
	err := loadCABundle(mapCertPools, defCleanedCAPath, DefaultHTTPClientConfig.CABundlePath)
	if err != nil {
		return nil, err
	}

	for _, serviceConfig := range b.config.ServiceConfigs {
		caBundlePath := serviceConfig.MergedComponentConfigs().Client.CABundlePath
		cleanedCAPath := resolveCAPath(b.GetConfigPath(), caBundlePath)
		err = loadCABundle(mapCertPools, cleanedCAPath, caBundlePath)
		if err != nil {
			return nil, err
		}
	}

	buildClientFn := func(mc ClientConfig) apiclient.RetryClient {
		return createHTTPClient(mc, mapCertPools, rbfn)
	}

	return buildClientFn, nil
}

// Load loads the config data
func (b *defaultConfigBuilder) Load(path string) (*os.File, error) {
	log.Trace("Loading config file: " + path)
	b.configPath = path

	file, err := os.Open(path)
	if err != nil {
		msg := &cnErrors.ErrorLog{
			RootCause: "Error opening config file " + path,
			Err:       err,
		}
		return nil, msg
	}

	return file, err
}

// Read parses the JSON data and creates mergedComponentConfigs
// which are the merge of DefaultComponentConfigs and
// serviceConfig.ComponentConfigOverrides
func (b *defaultConfigBuilder) Read(configData io.Reader) error {
	log.Trace("Reading config data")

	configuration, errs := buildInitialConfig(configData)
	if errs != nil {
		return errs
	}

	// now populate mergedComponentConfigs using serviceConfig and defaults
	mergeError := mergeComponentConfigsForAllServices(configuration) // updates in place
	if mergeError != nil {
		return cnErrors.WithErrorAndCause(mergeError, "Error merging component configs")
	}

	NewHashCode(configuration.Hash)
	b.config = configuration
	return nil
}

func buildInitialConfig(configData io.Reader) (*Config, error) {
	theBytes, readerError := ioutil.ReadAll(configData)
	if readerError != nil {
		return nil, cnErrors.WithErrorAndCause(readerError, "Error reading config data")
	}

	byteReader := bytes.NewReader(theBytes)

	c := &Config{}
	decoder := json.NewDecoder(byteReader)
	decoder.DisallowUnknownFields()
	decoderError := decoder.Decode(&c)
	if decoderError != nil {
		return nil, cnErrors.WithErrorAndCause(decoderError, "Error decoding config data")
	}
	c.Hash = fmt.Sprintf("%x", md5.Sum(theBytes))

	return c, nil
}

// LoadServiceAuthKeys attempts to get an auth key from the keyGetter, using the the provided client for communication,
// for each service config that requires auth to be used.
func (b *defaultConfigBuilder) LoadServiceAuthKeys(keyGetter AuthKeyGetter, client apiclient.RetryClient) []error {
	log.Trace("Loading auth keys")
	errs := make([]error, 0)
	var err error

	for name, serviceConfig := range b.config.ServiceConfigs {
		if serviceConfig.AuthRequired {
			serviceConfig.AuthKey, err = keyGetter.GetServiceKey(serviceConfig, client)

			if err != nil {
				errs = append(errs, &cnErrors.ErrorLog{
					RootCause: "Error retrieving auth key for " + name + ":",
					Err:       err,
				})
			} else if serviceConfig.AuthKey == "" {
				errs = append(errs, errors.New("Empty auth key for "+name))
			}
		}
	}

	return errs
}

// mergeComponentConfigsForAllServices populates mergedComponentConfigs
// using serviceConfig and defaults
func mergeComponentConfigsForAllServices(c *Config) error {
	defaultCompConfigs := c.DefaultComponentConfigs
	for k, serviceConfig := range c.ServiceConfigs {
		err := mergeCompConfigs(&serviceConfig.ComponentConfigOverrides, &defaultCompConfigs, &serviceConfig.mergedComponentConfigs)
		if err != nil {
			return cnErrors.WithErrorAndCause(err, "Error merging component config: "+k)
		}
	}
	return nil
}

// mergeCompConfigs merges ServiceComponentConfigOverrides and
// DefaultComponentConfigs. It starts by copying over the overrides
// and for any zero value fields, copies from the defaults
// and stores the values in the mergedCC. If serviceCCO or defaultCC
// is nil it is skipped in the copy. Passing a nil mergedCC target
// returns an error
func mergeCompConfigs(serviceCCO *ComponentConfigs, defaultCC *ComponentConfigs, mergedCC *ComponentConfigs) error {
	if mergedCC == nil {
		return errors.New("nil pointer passed for mergedCC")
	}

	if serviceCCO != nil {
		err := copier.Copy(mergedCC, serviceCCO)
		if err != nil {
			return err
		}
	}

	if defaultCC != nil {
		// set any zero values with defaults
		err := mergo.Merge(mergedCC, defaultCC)
		if err != nil {
			return err
		}
	}
	return nil
}

func createHTTPClient(mc ClientConfig, mapCertPools bundleMap, rbfn RetryClientBuilderFn) apiclient.RetryClient {
	// mc mergedClient has already been merged from serviceCCO and defaultCC
	disableCompression := false
	if mc.DisableCompression == True {
		disableCompression = true
	}

	tlsConfig := &tls.Config{}
	if mc.InsecureSkipVerify == True {
		tlsConfig.InsecureSkipVerify = true
	} else { // not skipping, so set cert pool
		certPool, ok := mapCertPools[mc.CABundlePath]
		if ok {
			tlsConfig.RootCAs = certPool
		}
	}

	baseClient := &http.Client{
		Timeout: time.Duration(mc.Timeout) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig:     tlsConfig,
			IdleConnTimeout:     time.Duration(mc.IdleConnTimeout) * time.Second,
			MaxIdleConnsPerHost: mc.MaxIdleConnsPerHost,
			MaxConnsPerHost:     mc.MaxConnsPerHost,
			DisableCompression:  disableCompression,
		},
	}

	retryClient := rbfn(mc.MaxRetries, baseClient)
	return retryClient
}

// resolveCAPath resolve relative to jsonPath and if cerPath is empty
// or resolvedPath is "." return empty string to signify no caBundlePath
func resolveCAPath(jsonPath string, certPath string) string {
	if certPath == "" {
		return ""
	}
	configDir := path.Dir(jsonPath)
	resolvedPath := compath.Resolve(configDir, certPath)
	if resolvedPath == "." { // not a valid CABundlePath
		return ""
	}
	return resolvedPath
}
