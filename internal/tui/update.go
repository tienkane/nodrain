package tui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.SetWidth(msg.Width)
		if m.phase == phaseTable {
			m.buildTable()
		}
		return m, nil

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		}
		switch m.phase {
		case phaseTable:
			return m.updateTableKeys(msg)
		case phaseAction:
			return m.updateActionKeys(msg)
		}
		return m, nil

	case progressMsg:
		m.stage = string(msg)
		return m, listenProgress(m.progressCh)

	case progressDoneMsg:
		return m, nil

	case scanDoneMsg:
		m.approvals = msg.approvals
		m.phase = phaseTable
		m.buildTable()
		return m, nil

	case errMsg:
		m.err = msg.err
		m.phase = phaseError
		return m, nil

	case wroteMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil // clear any earlier failed-write error
		m.wrotePath = msg.path
		m.wroteCount = msg.count
		return m, nil

	case spinner.TickMsg:
		if m.phase == phaseLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}
	return m, nil
}

func (m model) updateTableKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Toggle):
		if len(m.approvals) > 0 {
			id := m.approvals[m.table.Cursor()].ID()
			m.selected[id] = !m.selected[id]
			m.buildTable()
		}
		return m, nil
	case key.Matches(msg, keys.All):
		m.toggleAll()
		m.buildTable()
		return m, nil
	case key.Matches(msg, keys.Confirm):
		m.err = nil
		m.phase = phaseAction
		return m, nil
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) updateActionKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Back):
		m.phase = phaseTable
		m.wrotePath = ""
		m.err = nil
		return m, nil
	case key.Matches(msg, keys.Write):
		sel := m.selectedApprovals()
		if len(sel) == 0 {
			return m, nil
		}
		return m, m.writeBundleCmd(sel)
	}
	return m, nil
}
