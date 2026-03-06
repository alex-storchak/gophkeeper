package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

var (
	ErrEmptySessionSecret   = errors.New("config session.secret must not be empty")
	ErrEmptySessionFilePath = errors.New("config session.file path must not be empty")
)

func Load() (*Config, error) {
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.SetConfigName("client")

	applyDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if cfg.Session.Secret == "" {
		return nil, ErrEmptySessionSecret
	}
	if cfg.Session.File == "" {
		return nil, ErrEmptySessionFilePath
	}

	return &cfg, nil
}
