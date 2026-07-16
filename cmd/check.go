package cmd

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Zoxine/v2/internal/config"
	"github.com/Zoxine/v2/internal/pipeline"
)

var checkCmd = &cobra.Command{
	Use:   "check [file]",
	Short: "Validate proxies in a file and push working configs to GitHub",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadRuntimeConfig()
		if err != nil {
			return err
		}

		lines, err := readProxyFile(args[0])
		if err != nil {
			return err
		}

		runner := pipeline.New(cfg)
		result, err := runner.Run(context.Background(), lines)
		if err != nil {
			return err
		}

		fmt.Println(result.Message)
		return nil
	},
}

func init() {
	flags := checkCmd.Flags()
	flags.String("checker-bin", "", "path to v2ray-checker binary")
	flags.Int("checker-timeout", 0, "per-proxy timeout in seconds")
	flags.Int("checker-workers", 0, "checker worker count")
	flags.Bool("sort-speed", false, "sort results by speed")
	flags.String("github-token", "", "GitHub PAT")
	flags.String("github-owner", "", "GitHub repository owner")
	flags.String("github-repo", "", "GitHub repository name")
	flags.String("github-branch", "", "GitHub branch")

	_ = viper.BindPFlag("checker.binary_path", flags.Lookup("checker-bin"))
	_ = viper.BindPFlag("checker.timeout_seconds", flags.Lookup("checker-timeout"))
	_ = viper.BindPFlag("checker.workers", flags.Lookup("checker-workers"))
	_ = viper.BindPFlag("checker.sort_speed", flags.Lookup("sort-speed"))
	_ = viper.BindPFlag("github.token", flags.Lookup("github-token"))
	_ = viper.BindPFlag("github.owner", flags.Lookup("github-owner"))
	_ = viper.BindPFlag("github.repo", flags.Lookup("github-repo"))
	_ = viper.BindPFlag("github.branch", flags.Lookup("github-branch"))

	rootCmd.AddCommand(checkCmd)
}

func readProxyFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return lines, nil
}

func loadRuntimeConfig() (config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return cfg, err
	}
	if cfg.Verbose {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
	}
	return cfg, nil
}
