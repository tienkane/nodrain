// Package text holds string helpers shared across nodrain. Its job is to make
// untrusted, attacker-controlled strings — token symbols, remote spender labels,
// exploit names — safe to print to a terminal.
package text

import (
	"strings"
	"unicode"
)

// Sanitize strips characters that could corrupt or spoof a terminal when an
// untrusted string is rendered: C0 controls and DEL, the C1 control range
// (U+0080–U+009F, which includes the single-byte CSI introducer U+009B), and
// Unicode format characters (category Cf — bidirectional overrides such as
// U+202E and zero-width characters such as U+200B). Surrounding whitespace is
// trimmed. Every externally-sourced string that reaches the screen must pass
// through here.
func Sanitize(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case r < 0x20, r >= 0x7f && r <= 0x9f:
			return -1
		case unicode.Is(unicode.Cf, r):
			return -1
		default:
			return r
		}
	}, s)
	return strings.TrimSpace(s)
}
