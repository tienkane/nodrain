package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"

	"github.com/tienkane/nodrain/internal/chains"
	"github.com/tienkane/nodrain/internal/scanner"
)

// scanFlags are the inputs shared by the scan and export commands.
type scanFlags struct {
	chain    string
	address  string
	etherKey string
	rpc      string
	proxy    bool
	noLabels bool
	noRisk   bool
}

// bind registers the common flags on a command.
func (f *scanFlags) bind(cmd *cobra.Command) {
	fl := cmd.Flags()
	fl.StringVar(&f.chain, "chain", "ethereum", "chain to scan ("+chains.NamesCSV()+")")
	fl.StringVar(&f.address, "address", "", "wallet address to scan (or pass it as the first argument)")
	fl.StringVar(&f.etherKey, "etherscan-key", os.Getenv("ETHERSCAN_API_KEY"), "Etherscan V2 API key [env: ETHERSCAN_API_KEY]")
	fl.StringVar(&f.rpc, "rpc", os.Getenv("NODRAIN_RPC_URL"), "RPC endpoint for on-chain reads [env: NODRAIN_RPC_URL]")
	fl.BoolVar(&f.proxy, "proxy", false, "read on-chain state through the Etherscan proxy instead of an RPC node")
	fl.BoolVar(&f.noLabels, "no-labels", false, "skip remote spender-label lookups (whois.revoke.cash)")
	fl.BoolVar(&f.noRisk, "no-risk", false, "skip the approval-exploit-list risk check")
}

// options resolves the flags (and an optional positional address argument) into
// a scanner.Options, returning user-friendly errors.
func (f *scanFlags) options(args []string) (scanner.Options, error) {
	chain, ok := chains.Lookup(f.chain)
	if !ok {
		return scanner.Options{}, fmt.Errorf("unknown chain %q (supported: %s)", f.chain, chains.NamesCSV())
	}

	address := f.address
	if len(args) > 0 && args[0] != "" {
		address = args[0]
	}
	address = strings.TrimSpace(address)
	if address == "" {
		return scanner.Options{}, errors.New("a wallet address is required (pass it as an argument or with --address)")
	}
	if !common.IsHexAddress(address) {
		return scanner.Options{}, fmt.Errorf("invalid wallet address: %q", address)
	}

	if f.etherKey == "" {
		return scanner.Options{}, errors.New("an Etherscan API key is required: pass --etherscan-key or set ETHERSCAN_API_KEY (free key at https://etherscan.io/myapikey)")
	}

	return scanner.Options{
		Chain:        chain,
		Owner:        common.HexToAddress(address).Hex(),
		EtherscanKey: f.etherKey,
		RPCURL:       f.rpc,
		UseProxy:     f.proxy,
		Labels:       !f.noLabels,
		CheckRisk:    !f.noRisk,
	}, nil
}

// warnIfPaidChain prints a pre-flight heads-up when the chain is no longer on
// Etherscan's free API tier, so a plan error later is not a surprise.
func warnIfPaidChain(opts scanner.Options) {
	if !opts.Chain.EtherscanFree {
		fmt.Fprintf(os.Stderr, "note: %s is not on Etherscan's free API tier — a paid API key may be required.\n", opts.Chain.DisplayName)
	}
}
