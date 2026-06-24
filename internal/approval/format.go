package approval

import (
	"math/big"
	"strings"
)

// FormatUnits renders a raw integer token amount scaled by decimals into a
// human string with thousands separators, e.g. (12345600000, 6) -> "12,345.6".
// Trailing fractional zeros are trimmed; the fraction is capped at six places.
func FormatUnits(amount *big.Int, decimals uint8) string {
	if amount == nil || amount.Sign() == 0 {
		return "0"
	}
	if decimals == 0 {
		return groupThousands(amount.String())
	}

	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	intPart := new(big.Int)
	frac := new(big.Int)
	intPart.QuoRem(amount, scale, frac)

	out := groupThousands(intPart.String())
	if frac.Sign() == 0 {
		return out
	}

	// Left-pad the fraction to `decimals` digits, then trim to six places and
	// drop trailing zeros.
	fracStr := frac.String()
	if pad := int(decimals) - len(fracStr); pad > 0 {
		fracStr = strings.Repeat("0", pad) + fracStr
	}
	if len(fracStr) > 6 {
		fracStr = fracStr[:6]
	}
	fracStr = strings.TrimRight(fracStr, "0")
	if fracStr == "" {
		return out
	}
	return out + "." + fracStr
}

// groupThousands inserts commas into a non-negative integer string.
func groupThousands(s string) string {
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	n := len(s)
	if n <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	lead := n % 3
	if lead > 0 {
		b.WriteString(s[:lead])
		if n > lead {
			b.WriteByte(',')
		}
	}
	for i := lead; i < n; i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < n {
			b.WriteByte(',')
		}
	}
	return b.String()
}
