package config

import (
	"github.com/golang/glog"
	"github.com/spf13/viper"
)

// Configuration struct consisting configuration object
type Configuration struct {
	Server ServerConfiguration
}

// New create new instance of configuration object based on configuration file
func New() (*Configuration, error) {
	viper.SetConfigName("default")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	viper.SetConfigName(".env")
	if err := viper.MergeInConfig(); err != nil {
		glog.Warningf("Failed to load custom configuration file: %s", err)
	}

	cfg := new(Configuration)
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
