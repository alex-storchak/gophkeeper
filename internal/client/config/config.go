package config

type Config struct {
	ServerAddress string  `mapstructure:"server_address"`
	MaxRetries    int     `mapstructure:"max_retries"`
	TLS           TLS     `mapstructure:"tls"`
	Log           Log     `mapstructure:"log"`
	Session       Session `mapstructure:"session"`
}

type TLS struct {
	CACertFile string `mapstructure:"ca_cert_file"`
}

type Log struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

type Session struct {
	File   string `mapstructure:"file"`
	Secret string `mapstructure:"secret"`
}
