package model

import (
	"fmt"
	"strings"
	"time"
)

type ProxyURI struct {
	Raw      string
	Protocol string
	Host     string
	Port     string
	ID       string
	Label    string
}

func ParseProxyURI(raw string) ProxyURI {
	raw = strings.TrimSpace(raw)
	p := ProxyURI{Raw: raw}

	if idx := strings.Index(raw, "://"); idx > 0 {
		p.Protocol = strings.ToLower(raw[:idx])
	}

	if hash := strings.LastIndex(raw, "#"); hash >= 0 && hash < len(raw)-1 {
		p.Label = raw[hash+1:]
	}

	if p.Protocol == "vless" {
		p.parseVLESS()
	}

	return p
}

func (p *ProxyURI) parseVLESS() {
	rest := strings.TrimPrefix(p.Raw, "vless://")
	if hash := strings.Index(rest, "#"); hash >= 0 {
		rest = rest[:hash]
	}

	at := strings.LastIndex(rest, "@")
	if at < 0 {
		return
	}

	p.ID = rest[:at]
	hostPort := rest[at+1:]
	if q := strings.Index(hostPort, "?"); q >= 0 {
		hostPort = hostPort[:q]
	}

	p.Host, p.Port = splitHostPort(hostPort)
}

func splitHostPort(hostPort string) (string, string) {
	if strings.HasPrefix(hostPort, "[") {
		end := strings.Index(hostPort, "]")
		if end > 0 && end+1 < len(hostPort) && hostPort[end+1] == ':' {
			return hostPort[1:end], hostPort[end+2:]
		}
	}

	if colon := strings.LastIndex(hostPort, ":"); colon > 0 {
		return hostPort[:colon], hostPort[colon+1:]
	}

	return hostPort, ""
}

func (p ProxyURI) DedupeKey() string {
	if p.ID != "" && p.Host != "" {
		return p.Protocol + "|" + p.ID + "|" + p.Host + "|" + p.Port
	}
	return strings.TrimSpace(p.Raw)
}

func (p ProxyURI) WithLabel(label string) string {
	raw := strings.TrimSpace(p.Raw)
	if hash := strings.LastIndex(raw, "#"); hash >= 0 {
		return raw[:hash] + "#" + label
	}
	return raw + "#" + label
}

func IsAllowedProtocol(protocol string) bool {
	switch strings.ToLower(protocol) {
	case "vless", "vmess":
		return true
	default:
		return false
	}
}

func IsGenericLabel(label string) bool {
	label = strings.TrimSpace(strings.ToLower(label))
	if label == "" {
		return true
	}
	generic := []string{"default", "proxy", "config", "test", "node", "server"}
	for _, g := range generic {
		if label == g {
			return true
		}
	}
	return false
}

func NormalizeURIs(lines []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
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

func FilterByMode(lines []string, mode string) []string {
	if mode != "per-protocol" {
		return lines
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		p := ParseProxyURI(line)
		if IsAllowedProtocol(p.Protocol) {
			out = append(out, line)
		}
	}
	return out
}

func EnsureLabels(lines []string, prefix string) []string {
	date := time.Now().UTC().Format("20060102")
	out := make([]string, 0, len(lines))
	for i, line := range lines {
		p := ParseProxyURI(line)
		if IsGenericLabel(p.Label) {
			label := fmt.Sprintf("%s-%s-%d", prefix, date, i+1)
			out = append(out, p.WithLabel(label))
			continue
		}
		out = append(out, line)
	}
	return out
}
