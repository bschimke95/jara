package env

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines keyboard bindings for the application
type KeyMap struct {
	Quit       key.Binding
	Refresh    key.Binding
	SelectItem key.Binding
	Back       key.Binding
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view
// Implements key.Map interface
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Refresh, k.SelectItem, k.Back}
}

// FullHelp returns keybindings for the expanded help view
// Implements key.Map interface
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Quit, k.Refresh, k.SelectItem, k.Back},
	}
}

// DefaultKeyMap provides the default key bindings for the application
var DefaultKeyMap = KeyMap{
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	SelectItem: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("backspace"),
		key.WithHelp("backspace", "back"),
	),
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", "left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "right"),
	),
}
