package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	version = "dev"
)

var rootCmd = &cobra.Command{
	Use:   "v2",
	Short: "Telegram bot for validating and publishing V2Ray configs",
	Long:  "Collect vless:// and vmess:// URIs via Telegram, validate them with v2ray-checker, and append working configs to GitHub fixed content files.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize()

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ./config.yaml)")
	rootCmd.PersistentFlags().Bool("dry-run", false, "skip GitHub commits")
	rootCmd.PersistentFlags().Bool("verbose", false, "verbose logging")

	_ = viper.BindPFlag("dry_run", rootCmd.PersistentFlags().Lookup("dry-run"))
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

func initConfig() error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("V2AGENT")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
	_ = viper.BindEnv("telegram.token", "V2AGENT_TELEGRAM_TOKEN")
	_ = viper.BindEnv("github.token", "V2AGENT_GITHUB_TOKEN")

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok && cfgFile != "" {
			return fmt.Errorf("read config: %w", err)
		}
	}
	return nil
}

func setDefaults() {
	viper.SetDefault("telegram.flush_debounce_seconds", 10)
	viper.SetDefault("checker.binary_path", "v2ray-checker")
	viper.SetDefault("checker.timeout_seconds", 15)
	viper.SetDefault("checker.workers", 5)
	viper.SetDefault("checker.sort_speed", true)
	viper.SetDefault("github.owner", "Zoxine")
	viper.SetDefault("github.repo", "v2")
	viper.SetDefault("github.branch", "main")
	viper.SetDefault("github.commit_message", "feat: add static VLESS/VMESS configs to fixed content files")
	viper.SetDefault("github.files", []map[string]string{
		{"path": "src/contents/fixed-v2ray", "mode": "all"},
		{"path": "src/contents/fixed-filtered", "mode": "per-protocol"},
	})
}
