package ui

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap holds all keybindings for the application
type KeyMap struct {
	// Navigation
	Quit    key.Binding
	Refresh key.Binding
	Select  key.Binding
	Back    key.Binding

	// Actions
	Deploy  key.Binding
	Destroy key.Binding
	Scale   key.Binding
	Status  key.Binding

	// View
	ToggleDetails key.Binding
}

var DefaultKeyMap = KeyMap{
	// Navigation
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q/ctrl+c", "quit"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),

	// Actions
	Deploy: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "deploy"),
	),
	Destroy: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "destroy"),
	),
	Scale: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "scale"),
	),
	Status: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "status"),
	),

	// View
	ToggleDetails: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "toggle details"),
	),
}
