package text

import "testing"

func TestSanitize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "USDC", "USDC"},
		{"trims space", "  WETH  ", "WETH"},
		{"strips C0", "BAD\x07\x01TKN", "BADTKN"},
		{"strips DEL", "A\x7fB", "AB"},
		{"strips C1 CSI U+009B", "AB2J", "AB2J"},
		{"strips bidi override U+202E", "good‮EVIL", "goodEVIL"},
		{"strips zero-width U+200B", "a​b", "ab"},
		{"strips newline/CR", "line1\r\nline2", "line1line2"},
		{"keeps unicode letters", "Café", "Café"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Sanitize(tt.in); got != tt.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
