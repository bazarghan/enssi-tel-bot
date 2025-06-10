package config

import (
	"github.com/spf13/viper"
	"strings"
)

// Load initializes the configuration from file and environment variables.
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigName("config") // File name without extension
	v.SetConfigType("yaml")
	v.AddConfigPath(path)

	// Read from config file
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	// Read from environment variables
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
