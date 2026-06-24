package approval

import "math/big"

// MaxUint256 is type(uint256).max — the canonical "infinite" ERC-20 allowance.
var MaxUint256 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))

// MaxUint160 is type(uint160).max — Permit2's "unlimited" sentinel, a different
// number from the ERC-20 max.
var MaxUint160 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 160), big.NewInt(1))

// IsUnlimited reports whether an allowance should be treated as unlimited. It
// mirrors revoke.cash: an allowance larger than the token's total supply is
// effectively infinite, which catches type(uint256).max as well as the large
// round sentinels (2^96-1 and friends) some dapps grant. When totalSupply is
// unknown it falls back to an exact type(uint256).max check.
func IsUnlimited(allowance, totalSupply *big.Int) bool {
	if allowance == nil {
		return false
	}
	if totalSupply != nil && totalSupply.Sign() > 0 {
		return allowance.Cmp(totalSupply) > 0
	}
	return allowance.Cmp(MaxUint256) == 0
}

// IsActive reports whether an allowance is still worth surfacing. A historical
// Approval event may since have been spent down to zero or revoked, so only a
// positive live allowance counts.
func IsActive(allowance *big.Int) bool {
	return allowance != nil && allowance.Sign() > 0
}
