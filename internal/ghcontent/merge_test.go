package ghcontent

import (
	"strings"
	"testing"
)

func TestMergeAppendPreservesHeaderAndDedupes(t *testing.T) {
	existing := `#profile-title: base64:%TITLE%
#profile-update-interval: 1
vless://uuid1@1.1.1.1:443?security=tls#old
`
	newLines := []string{
		"vless://uuid1@1.1.1.1:443?security=tls#old",
		"vless://uuid2@2.2.2.2:443?security=tls#new",
	}

	merged, added := MergeAppend(existing, newLines)
	if added != 1 {
		t.Fatalf("expected 1 added line, got %d", added)
	}
	for _, want := range []string{
		"#profile-title: base64:%TITLE%",
		"vless://uuid1@1.1.1.1:443?security=tls#old",
		"vless://uuid2@2.2.2.2:443?security=tls#new",
	} {
		if !strings.Contains(merged, want) {
			t.Fatalf("missing %q in merged content:\n%s", want, merged)
		}
	}
}

func TestSplitHeaderAndConfigs(t *testing.T) {
	content := "# header\n# another\n\nvless://a\n"
	header, configs := SplitHeaderAndConfigs(content)
	if len(header) != 3 {
		t.Fatalf("expected 3 header lines, got %d: %#v", len(header), header)
	}
	if len(configs) != 1 || configs[0] != "vless://a" {
		t.Fatalf("unexpected configs: %#v", configs)
	}
}
