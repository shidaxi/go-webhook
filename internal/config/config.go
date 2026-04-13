package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// InitConfig initializes Viper with defaults, config file, and env overrides.
// If configPath is empty, it searches default paths.
// Returns the parsed AppConfig.
func InitConfig(configPath string) (AppConfig, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("admin.port", 9090)
	v.SetDefault("log.format", "json")
	v.SetDefault("rules.path", "configs/rules.yaml")

	// Environment variables
	v.SetEnvPrefix("GOWEBHOOK")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs/")
		v.AddConfigPath("$HOME/.go-webhook/")
		v.AddConfigPath("/etc/go-webhook/")
	}

	if err := v.ReadInConfig(); err != nil {
		// Config file not found is OK — we have defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok && configPath != "" {
			return AppConfig{}, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return AppConfig{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}
