package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/tienkane/nodrain/internal/approval"
	"github.com/tienkane/nodrain/internal/export"
)

func (m model) View() tea.View {
	var body string
	switch m.phase {
	case phaseLoading:
		body = m.loadingView()
	case phaseTable:
		body = m.tableView()
	case phaseAction:
		body = m.actionView()
	case phaseError:
		body = m.errorView()
	}

	v := tea.NewView(body)
	v.AltScreen = true
	return v
}

func (m model) banner() string {
	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		titleStyle.Render("nodrain"),
		"  ",
		subtle.Render(fmt.Sprintf("%s · %s", approval.ShortAddress(m.opts.Owner), m.opts.Chain.DisplayName)),
	)
}

func (m model) loadingView() string {
	return "\n" + lipgloss.JoinVertical(
		lipgloss.Left,
		m.banner(),
		"",
		fmt.Sprintf("%s %s", m.spinner.View(), m.stage),
		"",
		subtle.Render("scanning approval history and live allowances…"),
	)
}

func (m model) tableView() string {
	if len(m.approvals) == 0 {
		return "\n" + lipgloss.JoinVertical(
			lipgloss.Left,
			m.banner(),
			"",
			green.Render("✓ No active token approvals found."),
			"",
			subtle.Render("This wallet is not exposed by any ERC-20 approval on this chain."),
			"",
			footerStyle.Render("press q to quit"),
		)
	}

	var unlimited, malicious int
	for _, a := range m.approvals {
		switch a.Risk {
		case approval.RiskMalicious:
			malicious++
		case approval.RiskUnlimited:
			unlimited++
		}
	}
	summary := fmt.Sprintf("%d approvals", len(m.approvals))
	if unlimited > 0 {
		summary += "  ·  " + red.Render(fmt.Sprintf("%d unlimited", unlimited))
	}
	if malicious > 0 {
		summary += "  ·  " + red.Render(fmt.Sprintf("⚠ %d on exploit list", malicious))
	}
	if n := len(m.selectedApprovals()); n > 0 {
		summary += "  ·  " + green.Render(fmt.Sprintf("%d selected", n))
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.banner(),
		subtle.Render(summary),
		m.table.View(),
		footerStyle.Render(m.help.View(keys)),
	)
}

func (m model) actionView() string {
	sel := m.selectedApprovals()
	header := headerStyle.Render(fmt.Sprintf("Revoke plan — %d selected", len(sel)))

	var lines []string
	if len(sel) == 0 {
		lines = append(lines, subtle.Render("Nothing selected. Press esc, then space to tick the approvals to revoke."))
	} else {
		for _, a := range sel {
			amount := a.AmountDisplay()
			if a.Unlimited {
				amount = red.Render(amount)
			}
			lines = append(lines, fmt.Sprintf("  • %-10s → %-28s %s", truncate(a.Symbol, 10), truncate(a.SpenderDisplay(), 28), amount))
		}
	}

	revokeURL := export.RevokeCashURL(m.opts.Owner, m.opts.Chain.ID)
	parts := []string{
		header,
		"",
		subtle.Render("Open in the browser to revoke interactively:"),
		"  " + link.Render(revokeURL),
		"",
		strings.Join(lines, "\n"),
	}

	switch {
	case m.err != nil:
		parts = append(parts, "", errorStyle.Render("✗ "+m.err.Error()))
	case m.wrotePath != "":
		parts = append(parts, "", green.Render(fmt.Sprintf("✓ wrote %d unsigned transaction(s) to %s", m.wroteCount, m.wrotePath)))
		parts = append(parts, subtle.Render("  These are UNSIGNED — submit them from your own wallet."))
	}

	foot := "  " + subtle.Render("[w] write unsigned bundle   [esc] back   [q] quit")
	parts = append(parts, "", footerStyle.Render(foot))

	// Constrain the box to the terminal so the border and long URL wrap instead
	// of overflowing the right edge on narrow terminals.
	boxWidth := m.width - 4
	if boxWidth < 24 {
		boxWidth = 24
	}
	return "\n" + boxStyle.Width(boxWidth).Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func (m model) errorView() string {
	return "\n" + lipgloss.JoinVertical(
		lipgloss.Left,
		m.banner(),
		"",
		errorStyle.Render("✗ "+m.err.Error()),
		"",
		footerStyle.Render("press q to quit"),
	)
}
