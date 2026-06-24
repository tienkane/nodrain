// Package chains is the static registry of the EVM chains nodrain supports. The
// Etherscan V2 API addresses every chain through a single key plus a chainid
// query parameter, so a chain is fully described by its id, a default public
// RPC endpoint, and whether Etherscan's free tier still serves its logs.
package chains

import "strings"

// Chain describes one supported network.
type Chain struct {
	Name        string // canonical lowercase key used on the command line
	ID          int64  // EVM / Etherscan V2 chain id
	DisplayName string // human label shown in the UI
	DefaultRPC  string // public RPC used when --rpc is not provided
	// EtherscanFree reports whether Etherscan's free API tier still serves
	// getLogs for this chain. As of 2025-11-22 the free tier dropped Base,
	// Optimism and BNB Smart Chain; scanning those needs a paid key.
	EtherscanFree bool
}

// registry is ordered for stable presentation (mainnet first).
var registry = []Chain{
	{Name: "ethereum", ID: 1, DisplayName: "Ethereum", DefaultRPC: "https://ethereum-rpc.publicnode.com", EtherscanFree: true},
	{Name: "arbitrum", ID: 42161, DisplayName: "Arbitrum One", DefaultRPC: "https://arbitrum-one-rpc.publicnode.com", EtherscanFree: true},
	{Name: "polygon", ID: 137, DisplayName: "Polygon PoS", DefaultRPC: "https://polygon-bor-rpc.publicnode.com", EtherscanFree: true},
	{Name: "base", ID: 8453, DisplayName: "Base", DefaultRPC: "https://base-rpc.publicnode.com", EtherscanFree: false},
	{Name: "optimism", ID: 10, DisplayName: "OP Mainnet", DefaultRPC: "https://optimism-rpc.publicnode.com", EtherscanFree: false},
	{Name: "bsc", ID: 56, DisplayName: "BNB Smart Chain", DefaultRPC: "https://bsc-rpc.publicnode.com", EtherscanFree: false},
}

var byName = func() map[string]Chain {
	m := make(map[string]Chain, len(registry))
	for _, c := range registry {
		m[c.Name] = c
	}
	return m
}()

// Lookup resolves a chain by its canonical name (case-insensitive).
func Lookup(name string) (Chain, bool) {
	c, ok := byName[strings.ToLower(strings.TrimSpace(name))]
	return c, ok
}

// Names returns every supported chain name in presentation order.
func Names() []string {
	names := make([]string, len(registry))
	for i, c := range registry {
		names[i] = c.Name
	}
	return names
}

// NamesCSV is the comma-separated chain list used in flag help text.
func NamesCSV() string {
	return strings.Join(Names(), ", ")
}
