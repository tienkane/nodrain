package labels

import "strings"

// Label is a human description of a spender: a brand plus the specific contract.
type Label struct {
	Name  string `json:"name"`  // brand, e.g. "Uniswap"
	Label string `json:"label"` // specific contract, e.g. "Universal Router"
}

// crossChainSeed holds spenders deployed at the same address on every chain
// (deterministic or vanity deployments). Keys are lowercase addresses.
var crossChainSeed = lower(map[string]Label{
	"0x000000000022D473030F116dDEE9F6B43aC78BA3": {"Uniswap", "Permit2"},
	"0x111111125421cA6dc452d289314280a0f8842A65": {"1inch", "Aggregation Router v6"},
	"0x1111111254EEB25477B68fb85Ed929f73A960582": {"1inch", "Aggregation Router v5"},
	"0xDef1C0ded9bec7F1a1670819833240f027b25EfF": {"0x", "Exchange Proxy"},
	"0x1E0049783F008A0085193E00003D00cd54003c71": {"OpenSea", "Conduit"},
	"0x0000000000000068F116a894984e2DB1123eB395": {"OpenSea", "Seaport 1.6"},
	"0x00000000000000ADc04C56Bf30aC9d3c0aAF14dC": {"OpenSea", "Seaport 1.5"},
	"0xC92E8bdf79f0507f65a392b0ab4667716BFE0110": {"CoW Swap", "GPv2 Vault Relayer"},
})

// perChainSeed holds chain-specific spenders, keyed by chain id then lowercase
// address.
var perChainSeed = map[int64]map[string]Label{
	1: lower(map[string]Label{ // Ethereum mainnet
		"0x66a9893cC07D91D95644AEDD05D03f95e1dBA8Af": {"Uniswap", "Universal Router v2"},
		"0x3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD": {"Uniswap", "Universal Router"},
		"0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D": {"Uniswap", "Router v2"},
		"0xE592427A0AEce92De3Edee1F18E0157C05861564": {"Uniswap", "SwapRouter v3"},
		"0x68b3465833fb72A70ecDF485E0e4C7bD8665Fc45": {"Uniswap", "SwapRouter02 v3"},
		"0xC36442b4a4522E871399CD717aBDD847Ab11FE88": {"Uniswap", "v3 Position Manager"},
		"0xd9e1cE17f2641f24aE83637ab66a2cca9C378B9F": {"SushiSwap", "Router"},
		"0xDeF1C0ded9bec7F1a1670819833240f027b25EfF": {"0x", "Exchange Proxy"},
	}),
}

// seedLookup resolves a spender from the bundled seed maps. The address must be
// lowercase.
func seedLookup(chainID int64, lowerAddr string) (Label, bool) {
	if m, ok := perChainSeed[chainID]; ok {
		if l, ok := m[lowerAddr]; ok {
			return l, true
		}
	}
	if l, ok := crossChainSeed[lowerAddr]; ok {
		return l, true
	}
	return Label{}, false
}

// lower returns a copy of m with all keys lowercased.
func lower(m map[string]Label) map[string]Label {
	out := make(map[string]Label, len(m))
	for k, v := range m {
		out[strings.ToLower(k)] = v
	}
	return out
}
