package ui

import (
	"charm.land/bubbles/v2/key"
)

// KeyMap defines the global keybindings for jara.
type KeyMap struct {
	Quit            key.Binding
	Help            key.Binding
	Back            key.Binding
	Enter           key.Binding
	Command         key.Binding
	Filter          key.Binding
	Up              key.Binding
	Down            key.Binding
	PageUp          key.Binding
	PageDown        key.Binding
	Top             key.Binding
	Bottom          key.Binding
	CancelInput     key.Binding
	Tab             key.Binding
	ScaleUp         key.Binding
	ScaleDown       key.Binding
	Deploy          key.Binding // D: deploy a new application charm
	Relate          key.Binding // r: add a relation between applications
	DeleteRelation  key.Binding // D: remove a relation
	LogsJump        key.Binding // Shift+L: jump to logs with entity pre-filter
	LogsView        key.Binding // l: open logs keeping current filter
	ClearFilter     key.Binding // Shift+D: clear active log filter
	SearchOpen      key.Binding // /: open inline search (debug-log)
	SearchNext      key.Binding // n: next search match
	SearchPrev      key.Binding // N: previous search match
	FilterOpen      key.Binding // Shift+F: open filter modal (debug-log)
	UnitsNav        key.Binding // Shift+U: navigate to units view
	ApplicationsNav key.Binding // Shift+A: navigate to applications view
	RelationsNav    key.Binding // Shift+R: navigate to relations view
	SecretsNav      key.Binding // Shift+S: navigate to secrets view
	MachinesNav     key.Binding // Shift+M: navigate to machines view
	OffersNav       key.Binding // O: navigate to offers view
	StorageNav      key.Binding // T: navigate to storage view
	Decode          key.Binding // d: decode/reveal secret content
	ApplyFilter     key.Binding // Shift+F: apply filter in modal
	Right           key.Binding // l/right: move right in modal
	Left            key.Binding // h/left: move left in modal
	RunAction       key.Binding // a: run action on unit
	ConfigNav       key.Binding // C: view application config
	Yank            key.Binding // y: copy selected row to clipboard
	Inspect         key.Binding // i: show detail info for selected row
	EntitySwitch    key.Binding // s: switch entity context (e.g. filter by app)
}

// DefaultKeyMap returns the default vim-style keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Command: key.NewBinding(
			key.WithKeys(":"),
			key.WithHelp(":", "command"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/up", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/down", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("ctrl+u", "pgup"),
			key.WithHelp("C-u", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("ctrl+d", "pgdown"),
			key.WithHelp("C-d", "page down"),
		),
		Top: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "bottom"),
		),
		CancelInput: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch pane"),
		),
		ScaleUp: key.NewBinding(
			key.WithKeys("+"),
			key.WithHelp("+", "scale up"),
		),
		ScaleDown: key.NewBinding(
			key.WithKeys("-"),
			key.WithHelp("-", "scale down"),
		),
		Deploy: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "deploy"),
		),
		Relate: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "relate"),
		),
		DeleteRelation: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete relation"),
		),
		LogsJump: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "logs"),
		),
		LogsView: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "logs"),
		),
		ClearFilter: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "clear filter"),
		),
		SearchOpen: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		SearchNext: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next match"),
		),
		SearchPrev: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("N", "prev match"),
		),
		FilterOpen: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "filter"),
		),
		UnitsNav: key.NewBinding(
			key.WithKeys("U"),
			key.WithHelp("U", "units"),
		),
		ApplicationsNav: key.NewBinding(
			key.WithKeys("A"),
			key.WithHelp("A", "applications"),
		),
		RelationsNav: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "relations"),
		),
		SecretsNav: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "secrets"),
		),
		MachinesNav: key.NewBinding(
			key.WithKeys("M"),
			key.WithHelp("M", "machines"),
		),
		OffersNav: key.NewBinding(
			key.WithKeys("O"),
			key.WithHelp("O", "offers"),
		),
		StorageNav: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "storage"),
		),
		Decode: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "decode"),
		),
		ApplyFilter: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "apply"),
		),
		Right: key.NewBinding(
			key.WithKeys("l", "right"),
			key.WithHelp("l/→", "right"),
		),
		Left: key.NewBinding(
			key.WithKeys("h", "left"),
			key.WithHelp("h/←", "left"),
		),
		RunAction: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "run action"),
		),
		ConfigNav: key.NewBinding(
			key.WithKeys("C"),
			key.WithHelp("C", "config"),
		),
		Yank: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy"),
		),
		Inspect: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "info"),
		),
		EntitySwitch: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "switch entity"),
		),
	}
}

// ShortHelp returns the short help bindings.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Command, k.Filter, k.Back, k.Quit, k.Help}
}

// FullHelp returns the full help bindings grouped by category.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown, k.Top, k.Bottom},
		{k.Enter, k.Back, k.Command, k.Filter, k.Yank},
		{k.Quit, k.Help},
	}
}
