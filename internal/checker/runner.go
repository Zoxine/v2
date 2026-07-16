package checker

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Zoxine/v2/internal/config"
	"github.com/Zoxine/v2/internal/model"
)

type Result struct {
	Working []model.ProxyURI
	Total   int
	Failed  int
	Stderr  string
}

type Runner struct {
	cfg config.CheckerConfig
}

func NewRunner(cfg config.CheckerConfig) *Runner {
	return &Runner{cfg: cfg}
}

func (r *Runner) Run(ctx context.Context, inputFile string) (Result, error) {
	binary, err := exec.LookPath(r.cfg.BinaryPath)
	if err != nil {
		if filepath.IsAbs(r.cfg.BinaryPath) {
			if _, statErr := os.Stat(r.cfg.BinaryPath); statErr == nil {
				binary = r.cfg.BinaryPath
			} else {
				return Result{}, fmt.Errorf("checker binary not found: %s", r.cfg.BinaryPath)
			}
		} else {
			return Result{}, fmt.Errorf("checker binary not found on PATH: %s", r.cfg.BinaryPath)
		}
	}

	outFile, err := os.CreateTemp("", "v2-working-*.txt")
	if err != nil {
		return Result{}, fmt.Errorf("create temp output file: %w", err)
	}
	outPath := outFile.Name()
	outFile.Close()
	defer os.Remove(outPath)

	total, err := countLines(inputFile)
	if err != nil {
		return Result{}, err
	}

	args := []string{
		"-w", fmt.Sprintf("%d", r.cfg.Workers),
		"-t", fmt.Sprintf("%d", r.cfg.TimeoutSeconds),
		"-o", outPath,
	}
	if r.cfg.SortSpeed {
		args = append(args, "--sort-speed")
	}
	args = append(args, inputFile)

	timeout := time.Duration(r.cfg.TimeoutSeconds*total+60) * time.Second
	if timeout < 2*time.Minute {
		timeout = 2 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, binary, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			return Result{}, fmt.Errorf("checker timed out after %s", timeout)
		}
		if _, statErr := os.Stat(outPath); statErr != nil {
			return Result{}, fmt.Errorf("checker failed: %w\n%s", err, stderr.String())
		}
	}

	working, err := ParseOutputFile(outPath)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Working: working,
		Total:   total,
		Failed:  total - len(working),
		Stderr:  stderr.String(),
	}, nil
}

func countLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("open input file: %w", err)
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) != "" {
			count++
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("read input file: %w", err)
	}
	return count, nil
}

func ParseOutputFile(path string) ([]model.ProxyURI, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open checker output: %w", err)
	}
	defer file.Close()

	var out []model.ProxyURI
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		out = append(out, model.ParseProxyURI(line))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read checker output: %w", err)
	}
	return out, nil
}

func WriteTempInput(lines []string) (string, error) {
	file, err := os.CreateTemp("", "v2-proxies-*.txt")
	if err != nil {
		return "", fmt.Errorf("create temp input file: %w", err)
	}
	defer file.Close()

	for _, line := range lines {
		if _, err := fmt.Fprintln(file, line); err != nil {
			os.Remove(file.Name())
			return "", fmt.Errorf("write temp input file: %w", err)
		}
	}
	return file.Name(), nil
}
