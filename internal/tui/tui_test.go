package tui

import (
	"context"
	"math/big"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/tienkane/nodrain/internal/approval"
	"github.com/tienkane/nodrain/internal/chains"
	"github.com/tienkane/nodrain/internal/scanner"
)

func testModel() model {
	ch, _ := chains.Lookup("ethereum")
	opts := scanner.Options{Chain: ch, Owner: "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"}
	return newModel(context.Background(), opts, nil, nil)
}

func sampleApprovals() []approval.Approval {
	return []approval.Approval{
		{ChainID: 1, Token: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", Spender: "0x1111111254EEB25477B68fb85Ed929f73A960582", Amount: approval.MaxUint256, Symbol: "USDC", Decimals: 6, Unlimited: true, Risk: approval.RiskUnlimited, SpenderLabel: "1inch v5"},
		{ChainID: 1, Token: "0xdAC17F958D2ee523a2206206994597C13D831ec7", Spender: "0xE592427A0AEce92De3Edee1F18E0157C05861564", Amount: big.NewInt(5_000000), Symbol: "USDT", Decimals: 6, Risk: approval.RiskNone, SpenderLabel: "Uniswap v3"},
	}
}

func TestModelRendersAllPhases(t *testing.T) {
	m := testModel()

	// Resize then deliver the scan result; the table should be built.
	next, _ := m.Update(tea.WindowSizeMsg{Width: 110, Height: 30})
	m = next.(model)

	next, _ = m.Update(scanDoneMsg{approvals: sampleApprovals()})
	m = next.(model)
	if m.phase != phaseTable {
		t.Fatalf("phase = %d, want table", m.phase)
	}
	if got := len(m.table.Rows()); got != 2 {
		t.Fatalf("table has %d rows, want 2", got)
	}

	// The rows must actually RENDER, not just exist — the bubbles table hides
	// every row when its viewport width is 0.
	if tv := m.table.View(); !strings.Contains(tv, "USDC") || !strings.Contains(tv, "USDT") {
		t.Fatalf("table rows not rendered (viewport width bug?):\n%q", tv)
	}

	// Render each phase without panicking.
	_ = m.View()

	// Select all, then enter the action screen.
	m.toggleAll()
	if got := len(m.selectedApprovals()); got != 2 {
		t.Fatalf("toggleAll selected %d, want 2", got)
	}
	m.buildTable()
	m.phase = phaseAction
	_ = m.View()

	// Error phase.
	m.phase = phaseError
	m.err = context.Canceled
	_ = m.View()
}

func TestProgressUpdatesStage(t *testing.T) {
	m := testModel()
	next, _ := m.Update(progressMsg("Reading on-chain state…"))
	m = next.(model)
	if m.stage != "Reading on-chain state…" {
		t.Errorf("stage = %q", m.stage)
	}
}

func TestErrMsgSwitchesToErrorPhase(t *testing.T) {
	m := testModel()
	next, _ := m.Update(errMsg{err: context.DeadlineExceeded})
	m = next.(model)
	if m.phase != phaseError || m.err == nil {
		t.Errorf("errMsg did not switch to error phase")
	}
}
