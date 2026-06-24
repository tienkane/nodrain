package evm

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// TestLiveMulticall exercises the real Multicall3 path against Ethereum mainnet,
// including the bytes32 symbol decode (MKR). It is opt-in: set
// NODRAIN_INTEGRATION=1 to run it (it needs network access).
func TestLiveMulticall(t *testing.T) {
	if os.Getenv("NODRAIN_INTEGRATION") == "" {
		t.Skip("set NODRAIN_INTEGRATION=1 to run the live mainnet test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	r, err := NewRPCReader(ctx, "https://ethereum-rpc.publicnode.com")
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer r.Close()

	usdc := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	mkr := common.HexToAddress("0x9f8F72aA9304c8B593d555F12eF6589cC3A579A2") // bytes32 symbol
	weth := common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2")

	meta, err := BatchTokenMeta(ctx, r, []common.Address{usdc, mkr, weth})
	if err != nil {
		t.Fatalf("BatchTokenMeta: %v", err)
	}

	if m := meta[usdc]; m.Symbol != "USDC" || m.Decimals != 6 {
		t.Errorf("USDC meta = %+v, want symbol USDC decimals 6", m)
	}
	if m := meta[mkr]; m.Symbol != "MKR" || m.Decimals != 18 {
		t.Errorf("MKR meta = %+v, want symbol MKR decimals 18 (bytes32 path)", m)
	}
	if m := meta[weth]; m.Symbol != "WETH" || m.Decimals != 18 {
		t.Errorf("WETH meta = %+v, want symbol WETH decimals 18", m)
	}
	if meta[usdc].TotalSupply == nil || meta[usdc].TotalSupply.Sign() <= 0 {
		t.Errorf("USDC totalSupply not read")
	}
	t.Logf("live OK: USDC=%s(%d) MKR=%s(%d) WETH=%s(%d)",
		meta[usdc].Symbol, meta[usdc].Decimals, meta[mkr].Symbol, meta[mkr].Decimals, meta[weth].Symbol, meta[weth].Decimals)
}
