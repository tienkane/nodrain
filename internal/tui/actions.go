package tui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/tienkane/nodrain/internal/approval"
	"github.com/tienkane/nodrain/internal/export"
)

// buildTable (re)builds the table from the current approvals and selection,
// preserving the cursor position across rebuilds.
func (m *model) buildTable() {
	cols := []table.Column{
		{Title: "", Width: 3},
		{Title: "Token", Width: 12},
		{Title: "Spender", Width: 26},
		{Title: "Allowance", Width: 16},
		{Title: "Notes", Width: 22},
	}
	rows := make([]table.Row, len(m.approvals))
	for i, a := range m.approvals {
		check := "[ ]"
		if m.selected[a.ID()] {
			check = green.Render("[x]")
		}
		allow := a.AmountDisplay()
		if a.Unlimited {
			allow = red.Render(allow)
		} else {
			allow = yellow.Render(allow)
		}
		rows[i] = table.Row{check, truncate(a.Symbol, 12), truncate(a.SpenderDisplay(), 26), allow, noteFor(a)}
	}

	height := m.height - 9
	if height < 3 {
		height = 3
	}
	// Without an explicit width the table's internal viewport stays 0-wide and
	// renders every row as an empty string — only the header would show.
	width := m.width
	if width < 20 {
		width = 20
	}

	st := table.DefaultStyles()
	st.Header = st.Header.Bold(true).Foreground(lipgloss.Color("63"))

	cur := 0
	if m.table.Rows() != nil {
		cur = m.table.Cursor()
	}
	m.table = table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
		table.WithWidth(width),
		table.WithStyles(st),
	)
	if cur >= 0 && cur < len(rows) {
		m.table.SetCursor(cur)
	}
}

// noteFor renders the short risk/flag note shown in the table.
func noteFor(a approval.Approval) string {
	switch a.Risk {
	case approval.RiskMalicious:
		return red.Render("⚠ EXPLOIT")
	case approval.RiskUnlimited:
		if a.IsPermit2 {
			return yellow.Render("Permit2")
		}
		return ""
	default:
		return ""
	}
}

func (m *model) toggleAll() {
	allSelected := len(m.approvals) > 0
	for _, a := range m.approvals {
		if !m.selected[a.ID()] {
			allSelected = false
			break
		}
	}
	for _, a := range m.approvals {
		m.selected[a.ID()] = !allSelected
	}
}

// writeBundleCmd writes the unsigned revoke bundle for the selected approvals to
// a file in the working directory.
func (m model) writeBundleCmd(sel []approval.Approval) tea.Cmd {
	opts := m.opts
	return func() tea.Msg {
		b, err := export.Build(opts.Owner, opts.Chain.Name, opts.Chain.ID, sel)
		if err != nil {
			return wroteMsg{err: err}
		}
		data, err := b.JSON()
		if err != nil {
			return wroteMsg{err: err}
		}
		path := fmt.Sprintf("nodrain-bundle-%s-%s.json", opts.Chain.Name, shortWallet(opts.Owner))
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return wroteMsg{err: err}
		}
		return wroteMsg{path: path, count: len(sel)}
	}
}

func shortWallet(addr string) string {
	a := strings.TrimPrefix(addr, "0x")
	if len(a) < 8 {
		return strings.ToLower(a)
	}
	return strings.ToLower(a[:8])
}

func truncate(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w <= 1 {
		return string(r[:w])
	}
	return string(r[:w-1]) + "…"
}
