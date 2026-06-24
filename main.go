// Command nodrain is a terminal ERC-20 token-approval scanner. It finds a
// wallet's active approvals across EVM chains, flags unlimited allowances and
// known-risky spenders, and exports revoke.cash links plus unsigned
// approve(spender,0) calldata. It never holds a private key and never signs or
// broadcasts a transaction.
package main

import "github.com/tienkane/nodrain/internal/cli"

func main() {
	cli.Execute()
}
