package tui

import "charm.land/bubbles/v2/key"

type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Toggle  key.Binding
	All     key.Binding
	Confirm key.Binding
	Write   key.Binding
	Back    key.Binding
	Help    key.Binding
	Quit    key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Toggle, k.All, k.Confirm, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Toggle, k.All},
		{k.Confirm, k.Write, k.Back},
		{k.Help, k.Quit},
	}
}

var keys = keyMap{
	Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Toggle:  key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "select")),
	All:     key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "select all")),
	Confirm: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "review")),
	Write:   key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "write bundle")),
	Back:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}
