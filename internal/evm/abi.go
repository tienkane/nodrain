// Package evm is the read-only on-chain layer. It batches ERC-20 reads
// (allowance, symbol, decimals, totalSupply) and Permit2 lookups through
// Multicall3 in a handful of eth_calls, and builds the unsigned approve(spender,
// 0) calldata used to revoke. Every call is a view call routed through a
// pluggable Reader (a real RPC node or the Etherscan proxy) — nothing here ever
// signs or sends a transaction.
package evm

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const erc20ABIJSON = `[
  {"name":"allowance","type":"function","stateMutability":"view","inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"outputs":[{"name":"","type":"uint256"}]},
  {"name":"decimals","type":"function","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint8"}]},
  {"name":"symbol","type":"function","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"string"}]},
  {"name":"name","type":"function","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"string"}]},
  {"name":"totalSupply","type":"function","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
  {"name":"approve","type":"function","stateMutability":"nonpayable","inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"outputs":[{"name":"","type":"bool"}]}
]`

// aggregate3 is the only Multicall3 method nodrain needs. It is marked payable in
// the contract, so the ABI must say payable too or Pack mismatches the method —
// but it is invoked via eth_call with zero value, never signed.
const multicall3ABIJSON = `[
  {"name":"aggregate3","type":"function","stateMutability":"payable",
   "inputs":[{"name":"calls","type":"tuple[]","components":[
       {"name":"target","type":"address"},
       {"name":"allowFailure","type":"bool"},
       {"name":"callData","type":"bytes"}]}],
   "outputs":[{"name":"returnData","type":"tuple[]","components":[
       {"name":"success","type":"bool"},
       {"name":"returnData","type":"bytes"}]}]}
]`

// Parsed ABIs. These are constants, so a parse failure is a programming error
// and panics at startup.
var (
	erc20ABI      = mustParseABI(erc20ABIJSON)
	multicall3ABI = mustParseABI(multicall3ABIJSON)
)

func mustParseABI(s string) abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(s))
	if err != nil {
		panic("evm: invalid ABI: " + err.Error())
	}
	return parsed
}
