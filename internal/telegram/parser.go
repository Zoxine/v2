package telegram

import (
	"regexp"
)

var (
	vlessPattern = regexp.MustCompile(`vless://\S+`)
	vmessPattern = regexp.MustCompile(`vmess://\S+`)
)

func ExtractURIs(text string) []string {
	if text == "" {
		return nil
	}

	seen := make(map[string]struct{})
	var out []string
	for _, pattern := range []*regexp.Regexp{vlessPattern, vmessPattern} {
		for _, match := range pattern.FindAllString(text, -1) {
			if _, ok := seen[match]; ok {
				continue
			}
			seen[match] = struct{}{}
			out = append(out, match)
		}
	}
	return out
}
