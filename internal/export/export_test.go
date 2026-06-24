package export

import (
	"math/big"
	"strings"
	"testing"

	"github.com/tienkane/nodrain/internal/approval"
)

func TestRevokeCashURL(t *testing.T) {
	got := RevokeCashURL("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", 8453)
	want := "https://revoke.cash/address/0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045?chainId=8453"
	if got != want {
		t.Errorf("RevokeCashURL = %q, want %q", got, want)
	}
}

func TestBuild(t *testing.T) {
	approvals := []approval.Approval{{
		ChainID:      1,
		Owner:        "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
		Token:        "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
		Spender:      "0x1111111254EEB25477B68fb85Ed929f73A960582",
		Amount:       approval.MaxUint256,
		Symbol:       "USDC",
		Decimals:     6,
		Unlimited:    true,
		SpenderLabel: "Aggregation Router v5",
	}}
	b, err := Build("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", "ethereum", 1, approvals)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Transactions) != 1 {
		t.Fatalf("got %d txs, want 1", len(b.Transactions))
	}
	tx := b.Transactions[0]
	if tx.Value != "0x0" {
		t.Errorf("value = %q, want 0x0", tx.Value)
	}
	if tx.To != approvals[0].Token {
		t.Errorf("to = %q, want token %q", tx.To, approvals[0].Token)
	}
	// approve(address,uint256) selector + zero amount.
	if !strings.HasPrefix(tx.Data, "0x095ea7b3") {
		t.Errorf("data does not start with approve selector: %s", tx.Data)
	}
	if !strings.HasSuffix(tx.Data, strings.Repeat("0", 64)) {
		t.Errorf("data amount word is not zero: %s", tx.Data)
	}
	if !strings.Contains(b.Note, "UNSIGNED") {
		t.Error("note should state the txs are unsigned")
	}
	_ = big.NewInt(0)
}
