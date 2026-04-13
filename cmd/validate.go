package cmd

import (
	"fmt"
	"os"

	"github.com/shidaxi/go-webhook/internal/config"
	"github.com/shidaxi/go-webhook/internal/engine"
	"github.com/spf13/cobra"
)

var rulesPath string

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate rules configuration file",
	Long:  "Parses and compiles all rules to check for errors without starting the server.",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := rulesPath
		if path == "" {
			// Try to load from config
			cfg, err := config.InitConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			path = cfg.Rules.Path
		}

		rules, err := config.LoadRulesFromFile(path)
		if err != nil {
			return fmt.Errorf("failed to load rules: %w", err)
		}

		compiled := engine.CompileRules(rules)

		hasErrors := false
		for _, cr := range compiled {
			if cr.CompileError != nil {
				fmt.Fprintf(os.Stderr, "FAIL: rule %q — %v\n", cr.Rule.Name, cr.CompileError)
				hasErrors = true
			} else {
				fmt.Printf("OK:   rule %q\n", cr.Rule.Name)
			}
		}

		if hasErrors {
			return fmt.Errorf("validation failed: one or more rules have compile errors")
		}

		fmt.Printf("\nAll %d rules validated successfully.\n", len(compiled))
		return nil
	},
}

func init() {
	validateCmd.Flags().StringVar(&rulesPath, "rules", "", "rules file path (overrides config)")
	rootCmd.AddCommand(validateCmd)
}
