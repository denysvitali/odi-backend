package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Init initializes Viper configuration with the ODI_ environment variable prefix
func Init() {
	viper.SetEnvPrefix("ODI")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()
}

// BindEnv binds an environment variable to a flag name
func BindEnv(flagName string, envVars ...string) {
	allVars := append([]string{flagName}, envVars...)
	_ = viper.BindEnv(allVars...)
}

// GetString returns a string config value
func GetString(key string) string {
	return viper.GetString(key)
}

// GetBool returns a boolean config value
func GetBool(key string) bool {
	return viper.GetBool(key)
}

// GetInt returns an integer config value
func GetInt(key string) int {
	return viper.GetInt(key)
}

// GetFloat64 returns a float64 config value
func GetFloat64(key string) float64 {
	return viper.GetFloat64(key)
}
