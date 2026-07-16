package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Zoxine/v2/internal/config"
)

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Write a starter config.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		target := "config.yaml"
		if cfgFile != "" {
			target = cfgFile
		}
		if _, err := os.Stat(target); err == nil {
			return fmt.Errorf("%s already exists", target)
		}
		if err := os.WriteFile(target, []byte(config.ExampleYAML()), 0o600); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
		cmd.Printf("Wrote %s\n", target)
		return nil
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration helpers",
}

func init() {
	configCmd.AddCommand(configInitCmd)
	rootCmd.AddCommand(configCmd)
}
