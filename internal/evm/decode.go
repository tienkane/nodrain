package evm

import (
	"math/big"
	"strings"

	"github.com/tienkane/nodrain/internal/text"
)

// DecodeSymbol reads an ERC-20 symbol()/name() return value, which is usually
// an ABI-encoded string but is a raw bytes32 on a few legacy tokens (MKR being
// the canonical example). It discriminates by length: exactly 32 bytes is
// bytes32 (trailing NULs trimmed); a longer payload carries the offset+length
// header of an ABI string.
func DecodeSymbol(out []byte) string {
	switch {
	case len(out) == 0:
		return ""
	case len(out) == 32:
		return text.Sanitize(strings.TrimRight(string(out), "\x00"))
	default:
		vals, err := erc20ABI.Unpack("symbol", out)
		if err == nil && len(vals) > 0 {
			if s, ok := vals[0].(string); ok {
				return text.Sanitize(s)
			}
		}
		// Fall back to treating it as padded bytes.
		return text.Sanitize(strings.TrimRight(string(out), "\x00"))
	}
}

// DecodeUint8 reads a uint8 (e.g. decimals) from a 32-byte word, where the value
// is right-aligned in the final byte.
func DecodeUint8(out []byte) (uint8, bool) {
	if len(out) < 32 {
		return 0, false
	}
	return out[31], true
}

// DecodeUint256 reads a uint256 from the first 32-byte word.
func DecodeUint256(out []byte) (*big.Int, bool) {
	if len(out) < 32 {
		return nil, false
	}
	return new(big.Int).SetBytes(out[:32]), true
}
