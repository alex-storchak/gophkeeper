package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

var (
	ErrEmptyAuthJWTSecret = errors.New("config auth.jwt_secret must not be empty")
	ErrEmptyDBDSN         = errors.New("config db.dsn must not be empty")
)

func Load() (*Config, error) {
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.SetConfigName("server")

	applyDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	if cfg.Auth.JWTSecret == "" {
		return nil, ErrEmptyAuthJWTSecret
	}

	if cfg.DB.DSN == "" {
		return nil, ErrEmptyDBDSN
	}

	return &cfg, nil
}
