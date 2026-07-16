package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Zoxine/v2/internal/telegram"
)

var botCmd = &cobra.Command{
	Use:   "bot",
	Short: "Telegram bot commands",
}

var botStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Telegram listener",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadRuntimeConfig()
		if err != nil {
			return err
		}
		if cfg.Telegram.Token == "" {
			return fmt.Errorf("telegram token is required (config or V2AGENT_TELEGRAM_TOKEN)")
		}

		bot, err := telegram.NewBot(cfg)
		if err != nil {
			return err
		}

		slog.Info("starting bot", "github_repo", fmt.Sprintf("%s/%s", cfg.GitHub.Owner, cfg.GitHub.Repo), "dry_run", cfg.DryRun)
		return bot.Run(context.Background())
	},
}

func init() {
	flags := botStartCmd.Flags()
	flags.String("telegram-token", "", "Telegram bot token")
	flags.String("allowed-user-ids", "", "Comma-separated Telegram user IDs")
	flags.String("checker-bin", "", "path to v2ray-checker binary")
	flags.Int("checker-timeout", 0, "per-proxy timeout in seconds")
	flags.Int("checker-workers", 0, "checker worker count")
	flags.Bool("sort-speed", false, "sort results by speed")
	flags.Int("flush-debounce", 0, "seconds of inactivity before auto-submit")
	flags.String("github-token", "", "GitHub PAT")
	flags.String("github-owner", "", "GitHub repository owner")
	flags.String("github-repo", "", "GitHub repository name")
	flags.String("github-branch", "", "GitHub branch")

	_ = viper.BindPFlag("telegram.token", flags.Lookup("telegram-token"))
	_ = viper.BindPFlag("checker.binary_path", flags.Lookup("checker-bin"))
	_ = viper.BindPFlag("checker.timeout_seconds", flags.Lookup("checker-timeout"))
	_ = viper.BindPFlag("checker.workers", flags.Lookup("checker-workers"))
	_ = viper.BindPFlag("checker.sort_speed", flags.Lookup("sort-speed"))
	_ = viper.BindPFlag("telegram.flush_debounce_seconds", flags.Lookup("flush-debounce"))
	_ = viper.BindPFlag("github.token", flags.Lookup("github-token"))
	_ = viper.BindPFlag("github.owner", flags.Lookup("github-owner"))
	_ = viper.BindPFlag("github.repo", flags.Lookup("github-repo"))
	_ = viper.BindPFlag("github.branch", flags.Lookup("github-branch"))

	botStartCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		rawIDs, _ := cmd.Flags().GetString("allowed-user-ids")
		if rawIDs != "" {
			ids, err := parseUserIDs(rawIDs)
			if err != nil {
				return err
			}
			viper.Set("telegram.allowed_user_ids", ids)
		}
		return nil
	}

	botCmd.AddCommand(botStartCmd)
	rootCmd.AddCommand(botCmd)
}

func parseUserIDs(raw string) ([]int64, error) {
	parts := strings.Split(raw, ",")
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid user id %q: %w", part, err)
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("allowed-user-ids must contain at least one id")
	}
	return ids, nil
}
