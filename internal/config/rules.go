package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const DefaultTimeout = 10 * time.Second

// LoadRulesFromFile reads and parses a rules YAML file.
// It applies defaults for missing method and timeout.
func LoadRulesFromFile(path string) ([]Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules file %s: %w", path, err)
	}

	var rf RulesFile
	if err := yaml.Unmarshal(data, &rf); err != nil {
		return nil, fmt.Errorf("failed to parse rules file %s: %w", path, err)
	}

	rules := make([]Rule, 0, len(rf.Rules))
	for _, r := range rf.Rules {
		if r.Target.Method == "" {
			r.Target.Method = "POST"
		}
		if r.Target.Timeout == 0 {
			r.Target.Timeout = DefaultTimeout
		}
		rules = append(rules, r)
	}

	return rules, nil
}
