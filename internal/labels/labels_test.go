package labels

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestSeedLookup(t *testing.T) {
	// Permit2 is cross-chain, so it resolves on any chain.
	if l, ok := seedLookup(137, strings.ToLower("0x000000000022D473030F116dDEE9F6B43aC78BA3")); !ok || l.Name != "Uniswap" {
		t.Errorf("Permit2 seed lookup failed: %+v ok=%v", l, ok)
	}
	// A Uniswap v2 router is Ethereum-only.
	if l, ok := seedLookup(1, strings.ToLower("0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D")); !ok || l.Name != "Uniswap" {
		t.Errorf("Uniswap v2 seed lookup failed: %+v ok=%v", l, ok)
	}
	// Unknown spender.
	if _, ok := seedLookup(1, "0x0000000000000000000000000000000000000001"); ok {
		t.Error("unknown spender should not resolve")
	}
}

// TestRiskKeyCanonicalization proves the exploit-list insert and lookup keys
// agree regardless of the address form the upstream list happens to use.
func TestRiskKeyCanonicalization(t *testing.T) {
	const exploit = "Heist 2099"
	db := &RiskDB{byKey: map[string]string{}}

	// Insert with a no-0x-prefix address (the form the old code mishandled).
	db.add(cachedExploit{ChainID: 1, Address: "1111111254EEB25477B68fb85Ed929f73A960582", Name: exploit})

	// Look up by every case form — all must hit.
	for _, form := range []string{
		"0x1111111254EEB25477B68fb85Ed929f73A960582", // checksummed
		"0x1111111254eeb25477b68fb85ed929f73a960582", // lowercase
		"1111111254EEB25477B68fb85Ed929f73A960582",   // no prefix
	} {
		if name, ok := db.Lookup(1, common.HexToAddress(form)); !ok || name != exploit {
			t.Errorf("Lookup(%s) = %q ok=%v, want %q true", form, name, ok, exploit)
		}
	}

	// Wrong chain must miss.
	if _, ok := db.Lookup(137, common.HexToAddress("0x1111111254EEB25477B68fb85Ed929f73A960582")); ok {
		t.Error("exploit on chain 1 should not match on chain 137")
	}
}

func TestRiskAddSkipsMalformedAndSanitizesName(t *testing.T) {
	db := &RiskDB{byKey: map[string]string{}}

	// Malformed address is skipped, not inserted under a bogus key.
	db.add(cachedExploit{ChainID: 1, Address: "0x123", Name: "bad"})
	if len(db.byKey) != 0 {
		t.Errorf("malformed address was inserted: %v", db.byKey)
	}

	// A hostile exploit name is sanitized before it can ever be rendered.
	db.add(cachedExploit{ChainID: 1, Address: "0x1111111254EEB25477B68fb85Ed929f73A960582", Name: "Evil\x1b[2J\x07"})
	name, ok := db.Lookup(1, common.HexToAddress("0x1111111254EEB25477B68fb85Ed929f73A960582"))
	if !ok {
		t.Fatal("sanitized entry not found")
	}
	if strings.ContainsAny(name, "\x1b\x07") {
		t.Errorf("name was not sanitized: %q", name)
	}
}

func TestNilRiskDBLookup(t *testing.T) {
	var db *RiskDB
	if _, ok := db.Lookup(1, common.HexToAddress("0xdead")); ok {
		t.Error("nil RiskDB should never match")
	}
}
