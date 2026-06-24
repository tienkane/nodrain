// Package tui is the interactive front end: a Bubble Tea v2 program that streams
// scan progress behind a spinner, then presents a multi-select table of
// approvals and an action screen that writes an unsigned revoke bundle to disk.
package tui

import (
	"context"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/tienkane/nodrain/internal/approval"
	"github.com/tienkane/nodrain/internal/scanner"
)

type phase int

const (
	phaseLoading phase = iota
	phaseTable
	phaseAction
	phaseError
)

// scanResult carries the outcome of the background scan.
type scanResult struct {
	approvals []approval.Approval
	err       error
}

// Messages routed through Update.
type (
	progressMsg     string
	progressDoneMsg struct{}
	scanDoneMsg     struct{ approvals []approval.Approval }
	errMsg          struct{ err error }
	wroteMsg        struct {
		path  string
		count int
		err   error
	}
)

type model struct {
	ctx  context.Context
	opts scanner.Options

	phase   phase
	stage   string
	spinner spinner.Model
	table   table.Model
	help    help.Model

	approvals []approval.Approval
	selected  map[string]bool
	err       error

	wrotePath  string
	wroteCount int

	width, height int

	progressCh <-chan string
	resultCh   <-chan scanResult
}

func newModel(ctx context.Context, opts scanner.Options, progressCh <-chan string, resultCh <-chan scanResult) model {
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	sp.Style = accent
	h := help.New()
	return model{
		ctx:        ctx,
		opts:       opts,
		phase:      phaseLoading,
		stage:      "Starting…",
		spinner:    sp,
		help:       h,
		selected:   map[string]bool{},
		progressCh: progressCh,
		resultCh:   resultCh,
		width:      100,
		height:     24,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		listenProgress(m.progressCh),
		listenResult(m.resultCh),
	)
}

// listenProgress blocks for the next progress stage, re-armed after each one
// until the channel closes.
func listenProgress(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		stage, ok := <-ch
		if !ok {
			return progressDoneMsg{}
		}
		return progressMsg(stage)
	}
}

// listenResult blocks for the final scan result.
func listenResult(ch <-chan scanResult) tea.Cmd {
	return func() tea.Msg {
		r := <-ch
		if r.err != nil {
			return errMsg{r.err}
		}
		return scanDoneMsg{r.approvals}
	}
}

// selectedApprovals returns the approvals whose rows are ticked, preserving
// table order.
func (m model) selectedApprovals() []approval.Approval {
	var out []approval.Approval
	for _, a := range m.approvals {
		if m.selected[a.ID()] {
			out = append(out, a)
		}
	}
	return out
}
