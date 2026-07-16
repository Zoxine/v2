package ghcontent

import (
	"strings"

	"github.com/Zoxine/v2/internal/model"
)

func SplitHeaderAndConfigs(content string) (header []string, configs []string) {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	inHeader := true
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if inHeader && (trimmed == "" || strings.HasPrefix(trimmed, "#")) {
			header = append(header, line)
			continue
		}
		inHeader = false
		if trimmed == "" {
			continue
		}
		configs = append(configs, trimmed)
	}
	return header, configs
}

func MergeAppend(existingContent string, newLines []string) (string, int) {
	header, existing := SplitHeaderAndConfigs(existingContent)
	seen := make(map[string]struct{}, len(existing))
	for _, line := range existing {
		key := model.ParseProxyURI(line).DedupeKey()
		seen[key] = struct{}{}
	}

	added := make([]string, 0, len(newLines))
	for _, line := range newLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key := model.ParseProxyURI(line).DedupeKey()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		added = append(added, line)
	}

	if len(added) == 0 {
		return existingContent, 0
	}

	var builder strings.Builder
	for _, line := range header {
		builder.WriteString(line)
		builder.WriteByte('\n')
	}
	for _, line := range existing {
		builder.WriteString(line)
		builder.WriteByte('\n')
	}
	for _, line := range added {
		builder.WriteString(line)
		builder.WriteByte('\n')
	}
	return strings.TrimRight(builder.String(), "\n") + "\n", len(added)
}
