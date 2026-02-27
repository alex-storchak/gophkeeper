package config

import "github.com/spf13/viper"

const (
	DefServerAddress = "localhost:50051"
	DefMaxRetries    = 3

	DefSessionFile   = ".gophkeeper_session"
	DefSessionSecret = ""

	DefLogLevel = "info"
	DefLogFile  = ""

	DefTLSCaCertFile = "./cert/ca.crt"
)

func applyDefaults() {
	viper.SetDefault("server_address", DefServerAddress)
	viper.SetDefault("session.file", DefSessionFile)
	viper.SetDefault("session.secret", DefSessionSecret)
	viper.SetDefault("retry", DefMaxRetries)
	viper.SetDefault("log.level", DefLogLevel)
	viper.SetDefault("log.file", DefLogFile)
	viper.SetDefault("tls.ca_cert_file", DefTLSCaCertFile)
}
