// Package export turns selected approvals into inert, safe-to-share artifacts: a
// revoke.cash deep link for the wallet and one unsigned approve(spender,0)
// transaction per approval. Nothing here signs or broadcasts — the user submits
// the calldata from their own wallet, or revokes in the linked UI.
package export

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/ethereum/go-ethereum/common"

	"github.com/tienkane/nodrain/internal/approval"
	"github.com/tienkane/nodrain/internal/evm"
)

const safetyNote = "These transactions are UNSIGNED. nodrain never holds your keys. " +
	"Submit each from your own wallet, or open the revokeCashUrl to revoke in the browser. " +
	"To = the token contract; data calls approve(spender, 0)."

// UnsignedTx is an inert revoke transaction the user can submit themselves.
type UnsignedTx struct {
	ChainID int64  `json:"chainId"`
	To      string `json:"to"`    // token contract address
	Data    string `json:"data"`  // approve(spender, 0) calldata
	Value   string `json:"value"` // always "0x0"

	Token   string `json:"token"`
	Symbol  string `json:"symbol"`
	Spender string `json:"spender"`
	Label   string `json:"spenderLabel,omitempty"`
}

// Bundle is the full export payload for one wallet on one chain.
type Bundle struct {
	Wallet        string       `json:"wallet"`
	Chain         string       `json:"chain"`
	ChainID       int64        `json:"chainId"`
	RevokeCashURL string       `json:"revokeCashUrl"`
	Note          string       `json:"note"`
	Transactions  []UnsignedTx `json:"transactions"`
}

// RevokeCashURL is the revoke.cash page for a wallet's approvals on a chain.
// There is no per-token deep link; the user revokes in-app after landing here.
func RevokeCashURL(wallet string, chainID int64) string {
	u := url.URL{Scheme: "https", Host: "revoke.cash", Path: "/address/" + wallet}
	q := u.Query()
	q.Set("chainId", fmt.Sprintf("%d", chainID))
	u.RawQuery = q.Encode()
	return u.String()
}

// Build assembles an export bundle from the chosen approvals. All approvals must
// belong to the same wallet and chain.
func Build(wallet, chain string, chainID int64, approvals []approval.Approval) (Bundle, error) {
	txs := make([]UnsignedTx, 0, len(approvals))
	for _, a := range approvals {
		data, err := evm.RevokeCalldataHex(common.HexToAddress(a.Spender))
		if err != nil {
			return Bundle{}, fmt.Errorf("build calldata for %s: %w", a.Spender, err)
		}
		txs = append(txs, UnsignedTx{
			ChainID: chainID,
			To:      a.Token,
			Data:    data,
			Value:   "0x0",
			Token:   a.Token,
			Symbol:  a.Symbol,
			Spender: a.Spender,
			Label:   a.SpenderLabel,
		})
	}
	return Bundle{
		Wallet:        wallet,
		Chain:         chain,
		ChainID:       chainID,
		RevokeCashURL: RevokeCashURL(wallet, chainID),
		Note:          safetyNote,
		Transactions:  txs,
	}, nil
}

// JSON renders the bundle as indented JSON.
func (b Bundle) JSON() ([]byte, error) {
	return json.MarshalIndent(b, "", "  ")
}
