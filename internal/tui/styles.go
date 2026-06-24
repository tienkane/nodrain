package tui

import "charm.land/lipgloss/v2"

// Palette. Numeric ANSI codes keep colours readable across terminal themes.
var (
	red    = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)  // unlimited / danger
	yellow = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))            // caution
	green  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))            // selected check
	subtle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))           // muted text
	accent = lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true) // headings
	link   = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Underline(true)

	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Padding(0, 1)
	headerStyle = accent.MarginBottom(1)
	footerStyle = subtle.MarginTop(1)
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	boxStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("57")).Padding(0, 1)
)
