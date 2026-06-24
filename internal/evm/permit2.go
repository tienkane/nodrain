package evm

import "github.com/ethereum/go-ethereum/common"

// Permit2 is Uniswap's universal approval router, deployed at the same address
// on every EVM chain. A token approval whose spender is Permit2 is almost always
// unlimited, and the real per-dapp allowances live inside Permit2 itself — so
// revoking the ERC-20 → Permit2 approval does not clear already-signed permits.
// nodrain surfaces this with a warning; it does not yet enumerate the downstream
// permits.
var Permit2 = common.HexToAddress("0x000000000022D473030F116dDEE9F6B43aC78BA3")

// IsPermit2 reports whether addr is the canonical Permit2 contract.
func IsPermit2(addr common.Address) bool { return addr == Permit2 }
