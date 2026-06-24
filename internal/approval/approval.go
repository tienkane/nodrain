// Package approval holds the core domain types that flow through nodrain: a
// single token approval, its risk classification, and the helpers that render
// an allowance for humans. Nothing here touches the network — it is pure data
// so it is trivially testable.
package approval

import (
	"fmt"
	"math/big"
	"strings"
)

// RiskLevel ranks how dangerous an approval is, worst last so the zero value is
// "no special risk".
type RiskLevel int

const (
	RiskNone      RiskLevel = iota // a finite allowance to a recognised spender
	RiskUnlimited                  // an unlimited allowance
	RiskMalicious                  // spender appears on a known-exploit list
)

func (r RiskLevel) String() string {
	switch r {
	case RiskMalicious:
		return "malicious"
	case RiskUnlimited:
		return "unlimited"
	default:
		return "ok"
	}
}

// Approval is one live ERC-20 allowance: owner has approved spender to move up
// to Allowance units of Token. The on-chain allowance has already been verified
// (a historical Approval log alone is never enough).
type Approval struct {
	ChainID int64    `json:"chainId"`
	Owner   string   `json:"owner"`     // checksummed wallet address
	Token   string   `json:"token"`     // checksummed ERC-20 contract
	Spender string   `json:"spender"`   // checksummed approved spender
	Amount  *big.Int `json:"-"`         // raw on-chain allowance (wei-like units)
	Raw     string   `json:"allowance"` // decimal string of Amount, for JSON

	Symbol   string `json:"symbol"`   // token symbol, best effort
	Decimals uint8  `json:"decimals"` // token decimals, defaults to 18

	Unlimited bool      `json:"unlimited"`
	Risk      RiskLevel `json:"-"`
	RiskName  string    `json:"risk"`             // RiskLevel.String(), for JSON
	Reason    string    `json:"reason,omitempty"` // why it is risky, if known

	SpenderName  string `json:"spenderName,omitempty"`  // brand, e.g. "Uniswap"
	SpenderLabel string `json:"spenderLabel,omitempty"` // specific, e.g. "Universal Router"

	IsPermit2 bool `json:"isPermit2"` // spender is the canonical Permit2 contract

	LastTxHash string `json:"lastTxHash,omitempty"`
	LastBlock  uint64 `json:"lastBlock,omitempty"`
}

// ID is a stable key for an approval, used as the multi-select key in the TUI
// and for de-duplication. Addresses are lowercased so it is canonical.
func (a Approval) ID() string {
	return fmt.Sprintf("%d|%s|%s", a.ChainID, strings.ToLower(a.Token), strings.ToLower(a.Spender))
}

// SpenderDisplay is the friendliest available name for the spender, falling
// back to a shortened address when the spender is unknown.
func (a Approval) SpenderDisplay() string {
	switch {
	case a.SpenderLabel != "":
		return a.SpenderLabel
	case a.SpenderName != "":
		return a.SpenderName
	default:
		return ShortAddress(a.Spender)
	}
}

// AmountDisplay renders the allowance for a table cell: "Unlimited" for
// effectively-infinite approvals, otherwise the human token amount.
func (a Approval) AmountDisplay() string {
	if a.Unlimited {
		return "Unlimited"
	}
	return FormatUnits(a.Amount, a.Decimals)
}

// Finalize derives the JSON-friendly mirror fields from the computed state. Call
// it once an approval is fully classified, before serialisation.
func (a *Approval) Finalize() {
	if a.Amount != nil {
		a.Raw = a.Amount.String()
	}
	a.RiskName = a.Risk.String()
}

// ShortAddress abbreviates an address as 0xabcd…1234 for compact display.
func ShortAddress(addr string) string {
	if len(addr) <= 12 {
		return addr
	}
	return addr[:6] + "…" + addr[len(addr)-4:]
}
