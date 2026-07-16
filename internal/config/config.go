package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Telegram TelegramConfig `mapstructure:"telegram"`
	Checker  CheckerConfig  `mapstructure:"checker"`
	GitHub   GitHubConfig   `mapstructure:"github"`
	DryRun   bool           `mapstructure:"dry_run"`
	Verbose  bool           `mapstructure:"verbose"`
}

type TelegramConfig struct {
	Token                string  `mapstructure:"token"`
	AllowedUserIDs       []int64 `mapstructure:"allowed_user_ids"`
	FlushDebounceSeconds int     `mapstructure:"flush_debounce_seconds"`
}

type CheckerConfig struct {
	BinaryPath     string `mapstructure:"binary_path"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
	Workers        int    `mapstructure:"workers"`
	SortSpeed      bool   `mapstructure:"sort_speed"`
}

type GitHubConfig struct {
	Token         string           `mapstructure:"token"`
	Owner         string           `mapstructure:"owner"`
	Repo          string           `mapstructure:"repo"`
	Branch        string           `mapstructure:"branch"`
	Files         []GitHubFileSpec `mapstructure:"files"`
	CommitMessage string           `mapstructure:"commit_message"`
}

type GitHubFileSpec struct {
	Path string `mapstructure:"path"`
	Mode string `mapstructure:"mode"`
}

func Default() Config {
	return Config{
		Telegram: TelegramConfig{
			FlushDebounceSeconds: 10,
		},
		Checker: CheckerConfig{
			BinaryPath:     "v2ray-checker",
			TimeoutSeconds: 15,
			Workers:        5,
			SortSpeed:      true,
		},
		GitHub: GitHubConfig{
			Owner:  "Zoxine",
			Repo:   "v2",
			Branch: "main",
			Files: []GitHubFileSpec{
				{Path: "src/contents/fixed-v2ray", Mode: "all"},
				{Path: "src/contents/fixed-filtered", Mode: "per-protocol"},
			},
			CommitMessage: "feat: add static VLESS/VMESS configs to fixed content files",
		},
	}
}

func Load() (Config, error) {
	cfg := Default()

	if err := viper.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("decode config: %w", err)
	}

	cfg.Telegram.Token = expandEnv(cfg.Telegram.Token)
	cfg.GitHub.Token = expandEnv(cfg.GitHub.Token)

	if cfg.Checker.Workers <= 0 {
		cfg.Checker.Workers = 5
	}
	if cfg.Checker.TimeoutSeconds <= 0 {
		cfg.Checker.TimeoutSeconds = 15
	}
	if cfg.Telegram.FlushDebounceSeconds <= 0 {
		cfg.Telegram.FlushDebounceSeconds = 10
	}
	if cfg.GitHub.Owner == "" {
		cfg.GitHub.Owner = "Zoxine"
	}
	if cfg.GitHub.Repo == "" {
		cfg.GitHub.Repo = "v2"
	}
	if cfg.GitHub.Branch == "" {
		cfg.GitHub.Branch = "main"
	}
	if len(cfg.GitHub.Files) == 0 {
		cfg.GitHub.Files = Default().GitHub.Files
	}
	if cfg.GitHub.CommitMessage == "" {
		cfg.GitHub.CommitMessage = Default().GitHub.CommitMessage
	}

	return cfg, nil
}

func expandEnv(value string) string {
	if value == "" {
		return value
	}
	return os.Expand(value, func(key string) string {
		return os.Getenv(key)
	})
}

func ExampleYAML() string {
	return `telegram:
  token: "${V2AGENT_TELEGRAM_TOKEN}"
  allowed_user_ids: [123456789]
  flush_debounce_seconds: 10

checker:
  binary_path: "v2ray-checker"
  timeout_seconds: 15
  workers: 5
  sort_speed: true

github:
  token: "${V2AGENT_GITHUB_TOKEN}"
  owner: "Zoxine"
  repo: "v2"
  branch: "main"
  files:
    - path: "src/contents/fixed-v2ray"
      mode: "all"
    - path: "src/contents/fixed-filtered"
      mode: "per-protocol"
  commit_message: "feat: add static VLESS/VMESS configs to fixed content files"

dry_run: false
verbose: false
`
}

func AllowedPaths() map[string]struct{} {
	return map[string]struct{}{
		"src/contents/fixed-v2ray":      {},
		"src/contents/fixed-filtered":   {},
		"src/contents/fixed-v2ray-supersub": {},
		"src/contents/fixed-warp":       {},
	}
}

func ValidateGitHubPaths(files []GitHubFileSpec) error {
	allowed := AllowedPaths()
	for _, file := range files {
		path := strings.TrimSpace(file.Path)
		if path == "" {
			return fmt.Errorf("github file path must not be empty")
		}
		if strings.HasPrefix(path, "subscriptions/") {
			return fmt.Errorf("refusing to modify subscriptions path: %s", path)
		}
		if path == "appsettings.json" {
			return fmt.Errorf("refusing to modify appsettings.json")
		}
		if _, ok := allowed[path]; !ok {
			return fmt.Errorf("path %q is not in the allowed fixed-content list", path)
		}
	}
	return nil
}
