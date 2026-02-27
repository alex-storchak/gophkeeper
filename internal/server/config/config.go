package config

import "time"

type Config struct {
	GRPCAddress              string        `mapstructure:"grpc_address"`
	ShutdownWaitSecsDuration time.Duration `mapstructure:"shutdown_wait_secs_duration"`
	DB                       DB            `mapstructure:"db"`
	Auth                     Auth          `mapstructure:"auth"`
	TLS                      TLS           `mapstructure:"tls"`
	Log                      Log           `mapstructure:"log"`
}

type TLS struct {
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

type Log struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

type Auth struct {
	JWTSecret string        `mapstructure:"jwt_secret"`
	TokenTTL  time.Duration `mapstructure:"token_ttl"`
}

type DB struct {
	DSN            string `mapstructure:"dsn"`
	MigrationsPath string `mapstructure:"migrations_path"`
}
