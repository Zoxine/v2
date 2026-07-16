package model

import "testing"

func TestParseVLESS(t *testing.T) {
	raw := "vless://abc-uuid@example.com:443?security=tls#my-label"
	p := ParseProxyURI(raw)
	if p.Protocol != "vless" {
		t.Fatalf("protocol = %q", p.Protocol)
	}
	if p.ID != "abc-uuid" {
		t.Fatalf("id = %q", p.ID)
	}
	if p.Host != "example.com" || p.Port != "443" {
		t.Fatalf("host/port = %q:%q", p.Host, p.Port)
	}
	if p.Label != "my-label" {
		t.Fatalf("label = %q", p.Label)
	}
}

func TestEnsureLabels(t *testing.T) {
	lines := []string{"vless://abc@1.1.1.1:443?security=tls"}
	out := EnsureLabels(lines, "v2")
	if out[0] == lines[0] {
		t.Fatalf("expected generated label, got %q", out[0])
	}
}
