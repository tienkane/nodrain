package approval

import (
	"math/big"
	"testing"
)

func bi(s string) *big.Int {
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("bad int: " + s)
	}
	return n
}

func TestIsUnlimited(t *testing.T) {
	supply := bi("1000000000000000000000000") // 1M * 1e18
	tests := []struct {
		name      string
		allowance *big.Int
		supply    *big.Int
		want      bool
	}{
		{"max uint256, no supply", MaxUint256, nil, true},
		{"max uint256, with supply", MaxUint256, supply, true},
		{"above supply", bi("2000000000000000000000000"), supply, true},
		{"sentinel below max but above supply", bi("79228162514264337593543950335"), supply, true}, // 2^96-1
		{"finite below supply", bi("5000000000000000000"), supply, false},
		{"exact max with no supply only", bi("123"), nil, false},
		{"zero", big.NewInt(0), supply, false},
		{"nil allowance", nil, supply, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUnlimited(tt.allowance, tt.supply); got != tt.want {
				t.Errorf("IsUnlimited(%v, %v) = %v, want %v", tt.allowance, tt.supply, got, tt.want)
			}
		})
	}
}

func TestFormatUnits(t *testing.T) {
	tests := []struct {
		amount   string
		decimals uint8
		want     string
	}{
		{"0", 18, "0"},
		{"1000000", 6, "1"},
		{"12345600000", 6, "12,345.6"},
		{"1000000000000000000", 18, "1"},
		{"1500000000000000000", 18, "1.5"},
		{"1234567890000000000000", 18, "1,234.56789"},
		{"42", 0, "42"},
		{"1000000", 0, "1,000,000"},
		{"1", 18, "0.000000000000000001"[:8]}, // capped to 6 fractional places -> "0" since first 6 are zeros
	}
	// The last case documents the six-place cap: 1 wei rounds to "0".
	tests[len(tests)-1].want = "0"

	for _, tt := range tests {
		got := FormatUnits(bi(tt.amount), tt.decimals)
		if got != tt.want {
			t.Errorf("FormatUnits(%s, %d) = %q, want %q", tt.amount, tt.decimals, got, tt.want)
		}
	}
}

func TestID(t *testing.T) {
	a := Approval{ChainID: 1, Token: "0xAbC", Spender: "0xDeF"}
	if got, want := a.ID(), "1|0xabc|0xdef"; got != want {
		t.Errorf("ID() = %q, want %q", got, want)
	}
}

func TestShortAddress(t *testing.T) {
	if got := ShortAddress("0x1234567890abcdef1234567890abcdef12345678"); got != "0x1234…5678" {
		t.Errorf("ShortAddress = %q", got)
	}
}
