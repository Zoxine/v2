package telegram

import "testing"

func TestExtractURIs(t *testing.T) {
	text := `Here are configs:
vless://uuid@1.1.1.1:443?security=tls#label
and vmess://eyJ2IjoiMiJ9#vmess1
vless://uuid@1.1.1.1:443?security=tls#label
`
	uris := ExtractURIs(text)
	if len(uris) != 2 {
		t.Fatalf("expected 2 URIs, got %d: %#v", len(uris), uris)
	}
}
