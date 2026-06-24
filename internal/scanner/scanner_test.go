package scanner

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/tienkane/nodrain/internal/approval"
	"github.com/tienkane/nodrain/internal/etherscan"
	"github.com/tienkane/nodrain/internal/evm"
)

func padTopic(addr string) string {
	return etherscan.AddressTopic(addr)
}

func makeLog(token, spender string, block, logIdx int, tx string) etherscan.Log {
	return etherscan.Log{
		Address:         token,
		Topics:          []string{etherscan.ApprovalTopic0, padTopic("0xdead"), padTopic(spender)},
		Data:            "0x",
		BlockNumber:     "0x" + big.NewInt(int64(block)).Text(16),
		LogIndex:        "0x" + big.NewInt(int64(logIdx)).Text(16),
		TransactionHash: tx,
	}
}

func TestDedupKeepsLatest(t *testing.T) {
	token := "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
	spender := "0x1111111254EEB25477B68fb85Ed929f73A960582"
	other := "0xE592427A0AEce92De3Edee1F18E0157C05861564"

	logs := []etherscan.Log{
		makeLog(token, spender, 100, 1, "0xold"),
		makeLog(token, spender, 200, 3, "0xnew"), // latest for (token,spender)
		makeLog(token, other, 150, 0, "0xother"),
	}
	cands := dedupApprovals(logs)
	if len(cands) != 2 {
		t.Fatalf("got %d candidates, want 2", len(cands))
	}
	// Find the (token,spender) candidate and confirm it kept the newer tx.
	var found bool
	for _, c := range cands {
		if c.Spender == common.HexToAddress(spender) {
			found = true
			if c.TxHash != "0xnew" || c.Block != 200 {
				t.Errorf("dedup kept wrong log: tx=%s block=%d", c.TxHash, c.Block)
			}
		}
	}
	if !found {
		t.Error("expected (token,spender) candidate missing")
	}
}

func TestClassifyDropsZeroAndFlagsUnlimited(t *testing.T) {
	owner := common.HexToAddress("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	tokenA := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	tokenB := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	spender := common.HexToAddress("0x1111111254EEB25477B68fb85Ed929f73A960582")

	cands := []candidate{
		{Token: tokenA, Spender: spender, Block: 10},
		{Token: tokenB, Spender: spender, Block: 11},
	}
	meta := map[common.Address]evm.TokenMeta{
		tokenA: {Symbol: "USDC", Decimals: 6, Found: true},
		tokenB: {Symbol: "USDT", Decimals: 6, Found: true},
	}
	allowances := []*big.Int{approval.MaxUint256, big.NewInt(0)} // second is spent down

	got := classify(1, owner, cands, meta, allowances)
	if len(got) != 1 {
		t.Fatalf("got %d approvals, want 1 (zero allowance dropped)", len(got))
	}
	if !got[0].Unlimited {
		t.Error("max-uint256 allowance should be flagged unlimited")
	}
	if got[0].Symbol != "USDC" {
		t.Errorf("symbol = %q, want USDC", got[0].Symbol)
	}
}

func TestSortApprovalsWorstFirst(t *testing.T) {
	in := []approval.Approval{
		{Symbol: "AAA", Risk: approval.RiskNone},
		{Symbol: "BBB", Risk: approval.RiskMalicious},
		{Symbol: "CCC", Risk: approval.RiskUnlimited},
	}
	sortApprovals(in)
	if in[0].Risk != approval.RiskMalicious || in[1].Risk != approval.RiskUnlimited || in[2].Risk != approval.RiskNone {
		t.Errorf("sort order wrong: %v / %v / %v", in[0].Risk, in[1].Risk, in[2].Risk)
	}
}
