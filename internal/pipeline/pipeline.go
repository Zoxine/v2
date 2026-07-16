package pipeline

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Zoxine/v2/internal/checker"
	"github.com/Zoxine/v2/internal/config"
	"github.com/Zoxine/v2/internal/ghcontent"
)

type Result struct {
	Message string
	Check   checker.Result
	GitHub  []ghcontent.UpdateResult
}

type Runner struct {
	cfg     config.Config
	checker *checker.Runner
}

func New(cfg config.Config) *Runner {
	return &Runner{
		cfg:     cfg,
		checker: checker.NewRunner(cfg.Checker),
	}
}

func (r *Runner) Run(ctx context.Context, lines []string) (Result, error) {
	lines = normalizeInput(lines)
	if len(lines) == 0 {
		return Result{}, fmt.Errorf("no proxy URIs to check")
	}

	inputFile, err := checker.WriteTempInput(lines)
	if err != nil {
		return Result{}, err
	}
	defer os.Remove(inputFile)

	checkResult, err := r.checker.Run(ctx, inputFile)
	if err != nil {
		return Result{}, fmt.Errorf("checker: %w", err)
	}
	if len(checkResult.Working) == 0 {
		return Result{
			Message: fmt.Sprintf("Checked %d configs — 0 working, %d failed.", checkResult.Total, checkResult.Failed),
			Check:   checkResult,
		}, nil
	}

	workingLines := make([]string, 0, len(checkResult.Working))
	for _, proxy := range checkResult.Working {
		workingLines = append(workingLines, proxy.Raw)
	}

	ghClient, err := ghcontent.NewClient(r.cfg.GitHub, r.cfg.DryRun, r.cfg.Verbose)
	if err != nil {
		return Result{}, fmt.Errorf("github client: %w", err)
	}

	updates, err := ghClient.AppendWorkingConfigs(ctx, r.cfg.GitHub.Files, r.cfg.GitHub.CommitMessage, workingLines)
	if err != nil {
		return Result{}, fmt.Errorf("github update: %w", err)
	}

	message := formatSuccessMessage(r.cfg.GitHub.Owner, r.cfg.GitHub.Repo, checkResult, updates, r.cfg.DryRun)
	return Result{
		Message: message,
		Check:   checkResult,
		GitHub:  updates,
	}, nil
}

func normalizeInput(lines []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "vless://") && !strings.HasPrefix(line, "vmess://") {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		out = append(out, line)
	}
	return out
}

func formatSuccessMessage(owner, repo string, check checker.Result, updates []ghcontent.UpdateResult, dryRun bool) string {
	var b strings.Builder
	if dryRun {
		fmt.Fprintf(&b, "✅ Checked %d configs — %d working, %d failed (dry-run).\n",
			check.Total, len(check.Working), check.Failed)
	} else {
		fmt.Fprintf(&b, "✅ Checked %d configs — %d working, %d failed.\n",
			check.Total, len(check.Working), check.Failed)
	}

	if len(updates) > 0 {
		fmt.Fprintf(&b, "Pushed to %s/%s:\n", owner, repo)
	}
	var firstURL string
	for _, update := range updates {
		if update.AddedCount == 0 {
			fmt.Fprintf(&b, " • %s (no new configs)\n", update.Path)
			continue
		}
		sha := update.CommitSHA
		if len(sha) > 7 {
			sha = sha[:7]
		}
		if update.CommitSHA != "" {
			fmt.Fprintf(&b, " • %s (commit %s)\n", update.Path, sha)
		} else {
			fmt.Fprintf(&b, " • %s (%d added)\n", update.Path, update.AddedCount)
		}
		if firstURL == "" && update.CommitURL != "" {
			firstURL = update.CommitURL
		}
	}
	if firstURL != "" {
		b.WriteString(firstURL)
	}
	return strings.TrimSpace(b.String())
}
