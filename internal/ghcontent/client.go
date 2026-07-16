package ghcontent

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Zoxine/v2/internal/config"
	"github.com/Zoxine/v2/internal/model"
	"github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"
)

type Client struct {
	api     *github.Client
	owner   string
	repo    string
	branch  string
	dryRun  bool
	verbose bool
}

type UpdateResult struct {
	Path       string
	CommitSHA  string
	CommitURL  string
	AddedCount int
}

func NewClient(cfg config.GitHubConfig, dryRun, verbose bool) (*Client, error) {
	if err := config.ValidateGitHubPaths(cfg.Files); err != nil {
		return nil, err
	}
	if cfg.Token == "" && !dryRun {
		return nil, fmt.Errorf("github token is required")
	}

	var httpClient *http.Client
	if cfg.Token != "" {
		httpClient = oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: cfg.Token},
		))
	} else {
		httpClient = http.DefaultClient
	}

	return &Client{
		api:     github.NewClient(httpClient),
		owner:   cfg.Owner,
		repo:    cfg.Repo,
		branch:  cfg.Branch,
		dryRun:  dryRun,
		verbose: verbose,
	}, nil
}

func (c *Client) AppendWorkingConfigs(ctx context.Context, files []config.GitHubFileSpec, commitMessage string, working []string) ([]UpdateResult, error) {
	if len(working) == 0 {
		return nil, fmt.Errorf("no working configs to append")
	}

	labeled := model.EnsureLabels(model.NormalizeURIs(working), "v2")
	results := make([]UpdateResult, 0, len(files))

	for _, spec := range files {
		filtered := model.FilterByMode(labeled, spec.Mode)
		if len(filtered) == 0 {
			continue
		}

		result, err := c.updateFile(ctx, spec.Path, commitMessage, filtered)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no github files were updated")
	}
	return results, nil
}

func (c *Client) updateFile(ctx context.Context, path, commitMessage string, newLines []string) (UpdateResult, error) {
	current, sha, err := c.getFile(ctx, path)
	if err != nil {
		return UpdateResult{}, err
	}

	merged, added := MergeAppend(current, newLines)
	if added == 0 {
		return UpdateResult{Path: path, AddedCount: 0}, nil
	}

	if c.dryRun {
		if c.verbose {
			fmt.Printf("[dry-run] would append %d line(s) to %s\n", added, path)
		}
		return UpdateResult{Path: path, AddedCount: added}, nil
	}

	contentBytes := []byte(merged)
	branch := c.branch
	opts := &github.RepositoryContentFileOptions{
		Message: github.String(commitMessage),
		Content: contentBytes,
		Branch:  &branch,
	}
	if sha != "" {
		opts.SHA = github.String(sha)
	}

	var commit github.Commit
	err = withRetry(ctx, func() error {
		resp, _, reqErr := c.api.Repositories.UpdateFile(ctx, c.owner, c.repo, path, opts)
		if reqErr != nil {
			return reqErr
		}
		if resp != nil {
			commit = resp.Commit
		}
		return nil
	})
	if err != nil {
		return UpdateResult{}, fmt.Errorf("update %s: %w", path, err)
	}

	result := UpdateResult{Path: path, AddedCount: added}
	if commit.SHA != nil {
		result.CommitSHA = *commit.SHA
	}
	if commit.HTMLURL != nil {
		result.CommitURL = *commit.HTMLURL
	}
	return result, nil
}

func (c *Client) getFile(ctx context.Context, path string) (string, string, error) {
	ref := c.branch
	var fileContent *github.RepositoryContent
	err := withRetry(ctx, func() error {
		content, _, _, respErr := c.api.Repositories.GetContents(ctx, c.owner, c.repo, path, &github.RepositoryContentGetOptions{Ref: ref})
		if respErr != nil {
			return respErr
		}
		if content == nil {
			return fmt.Errorf("file not found: %s", path)
		}
		fileContent = content
		return nil
	})
	if err != nil {
		return "", "", fmt.Errorf("get %s: %w", path, err)
	}

	decoded, err := fileContent.GetContent()
	if err != nil {
		return "", "", fmt.Errorf("decode %s: %w", path, err)
	}

	sha := ""
	if fileContent.SHA != nil {
		sha = *fileContent.SHA
	}
	return decoded, sha, nil
}

func withRetry(ctx context.Context, fn func() error) error {
	backoff := []time.Duration{0, 2 * time.Second, 5 * time.Second, 15 * time.Second}
	var lastErr error
	for i, wait := range backoff {
		if wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if !isRetryable(lastErr) || i == len(backoff)-1 {
			return lastErr
		}
	}
	return lastErr
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "403") ||
		strings.Contains(msg, "502") ||
		strings.Contains(msg, "503") ||
		strings.Contains(msg, "504")
}
