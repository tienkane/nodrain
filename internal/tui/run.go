package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/tienkane/nodrain/internal/scanner"
)

// Run launches the interactive scanner. The scan runs on a background goroutine
// that streams progress and the final result back to the Bubble Tea program
// over channels; the program is cancelled (and the scan with it) when the user
// quits.
func Run(ctx context.Context, opts scanner.Options) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	progressCh := make(chan string, 16)
	resultCh := make(chan scanResult, 1)

	go func() {
		approvals, err := scanner.Scan(ctx, opts, func(stage string) {
			// Non-blocking: the UI showing progress is best effort and must
			// never stall the scan.
			select {
			case progressCh <- stage:
			default:
			}
		})
		close(progressCh)
		resultCh <- scanResult{approvals: approvals, err: err}
	}()

	final, err := tea.NewProgram(newModel(ctx, opts, progressCh, resultCh)).Run()
	if err != nil {
		return err
	}
	if fm, ok := final.(model); ok && fm.err != nil {
		return fm.err
	}
	return nil
}
