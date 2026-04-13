package config

import "time"

// AppConfig holds the top-level application configuration.
type AppConfig struct {
	Server ServerConfig `mapstructure:"server"`
	Admin  AdminConfig  `mapstructure:"admin"`
	Log    LogConfig    `mapstructure:"log"`
	Rules  RulesConfig  `mapstructure:"rules"`
}

// ServerConfig holds business server settings.
type ServerConfig struct {
	Port int `mapstructure:"port"`
}

// AdminConfig holds admin server settings.
type AdminConfig struct {
	Port int `mapstructure:"port"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Format string `mapstructure:"format"`
}

// RulesConfig holds rules file path.
type RulesConfig struct {
	Path string `mapstructure:"path"`
}

// RulesFile represents the top-level structure of rules.yaml.
type RulesFile struct {
	Rules []Rule `yaml:"rules"`
}

// Rule defines a single webhook forwarding rule.
type Rule struct {
	Name   string     `yaml:"name"`
	Match  string     `yaml:"match"`
	Target RuleTarget `yaml:"target"`
	Body   string     `yaml:"body"`
}

// RuleTarget defines the forwarding destination.
type RuleTarget struct {
	URL     string            `yaml:"url"`
	Method  string            `yaml:"method"`
	Timeout time.Duration     `yaml:"timeout"`
	Headers map[string]string `yaml:"headers"`
}

// DispatchResult holds the outcome of a single dispatch attempt.
type DispatchResult struct {
	RuleName   string
	TargetURL  string
	StatusCode int
	Success    bool
	Error      error
}
