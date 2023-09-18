package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	defaultEnvironment     = "local"
	defaultServiceVersion  = "dev"
	defaultRestPort        = "8080"
	defaultShutdownTimeout = 20 * time.Second

	configKeyEnvironment          = "environment"
	configKeyServiceName          = "service_name"
	configKeyServiceVersion       = "service_version"
	configKeyRestPort             = "rest_port"
	configKeyShutdownTimout       = "shutdown_timeout"
	configKeyFibonacciServiceUrl  = "fibonacci_service_url"
	configKeyOTELExporterEndpoint = "otel_exporter_otlp_endpoint"
)

func init() {
	viper.AutomaticEnv()

	if viper.GetString(configKeyEnvironment) == "" {
		viper.SetDefault(configKeyEnvironment, defaultEnvironment)
	}

	if viper.GetString(configKeyServiceVersion) == "" {
		viper.SetDefault(configKeyServiceVersion, defaultServiceVersion)
	}

	if viper.GetString(configKeyRestPort) == "" {
		viper.SetDefault(configKeyRestPort, defaultRestPort)
	}

	if viper.GetDuration(configKeyShutdownTimout) == 0 {
		viper.SetDefault(configKeyShutdownTimout, defaultShutdownTimeout)
	}
}

func mustGetString(key string) string {
	value := viper.GetString(key)
	if value == "" {
		panic(fmt.Sprintf("%q is not set", key))
	}

	return value
}

func GetEnvironment() string {
	return mustGetString(configKeyEnvironment)
}

func GetServiceName() string {
	return mustGetString(configKeyServiceName)
}

func GetServiceVersion() string {
	return strings.ToLower(viper.GetString(configKeyServiceVersion))
}

func GetRestPort() string {
	return mustGetString(configKeyRestPort)
}

func GetFibonacciServiceUrl() string {
	return mustGetString(configKeyFibonacciServiceUrl)
}

func GetShutdownTimeout() time.Duration {
	return viper.GetDuration(configKeyShutdownTimout)
}

func GetOTELExporterEndpointUrl() string {
	return mustGetString(configKeyOTELExporterEndpoint)
}
