package config

import (
	"time"

	"github.com/spf13/viper"
)

const (
	DefGRPCAddress              = ":50051"
	DefShutdownWaitSecsDuration = 10 * time.Second

	DefLogLevel = "info"
	DefLogFile  = ""

	DefTLSCertFile = "./cert/server.crt"
	DefTLSKeyFile  = "./cert/server.key"

	DefAuthTokenTTL = 30 * time.Minute
)

func applyDefaults() {
	viper.SetDefault("grpc_address", DefGRPCAddress)
	viper.SetDefault("shutdown_wait_secs_duration", DefShutdownWaitSecsDuration)
	viper.SetDefault("log.level", DefLogLevel)
	viper.SetDefault("log.file", DefLogFile)
	viper.SetDefault("tls.cert_file", DefTLSCertFile)
	viper.SetDefault("tls.key_file", DefTLSKeyFile)
	viper.SetDefault("auth.token_ttl", DefAuthTokenTTL)
}
