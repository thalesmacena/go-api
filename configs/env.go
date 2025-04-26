package configs

import (
	"github.com/spf13/viper"
)

type EnvConfig struct {
	ApplicationName string
	ContextPath     string
}

var Env *EnvConfig

func init() {
	viper.AutomaticEnv()

	Env = &EnvConfig{
		ApplicationName: viper.GetString("APPLICATION_NAME"),
		ContextPath:     getStringOrDefault("CONTEXT_PATH", "/go-http"),
	}
}

func getStringOrDefault(key, defaultValue string) string {
	value := viper.GetString(key)
	if value == "" {
		return defaultValue
	}
	return value
}
