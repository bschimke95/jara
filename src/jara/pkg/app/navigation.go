package app

import (
	"github.com/charmbracelet/bubbles/key"
)

// TOOD move this into key-map
// NavigationKeyMap holds all keybindings for navigation
type Navigation interface {
	Quit() key.Binding
	Refresh() key.Binding
	Select() key.Binding
	Back() key.Binding

	Up() key.Binding
	Down() key.Binding
	Left() key.Binding
	Right() key.Binding
}

type defaultNavigation struct {
	quit       key.Binding
	refresh    key.Binding
	selectItem key.Binding
	back       key.Binding
	up         key.Binding
	down       key.Binding
	left       key.Binding
	right      key.Binding
}

func (n *defaultNavigation) Quit() key.Binding {
	return n.quit
}

func (n *defaultNavigation) Refresh() key.Binding {
	return n.refresh
}

func (n *defaultNavigation) Select() key.Binding {
	return n.selectItem
}

func (n *defaultNavigation) Back() key.Binding {
	return n.back
}

func (n *defaultNavigation) Up() key.Binding {
	return n.up
}

func (n *defaultNavigation) Down() key.Binding {
	return n.down
}

func (n *defaultNavigation) Left() key.Binding {
	return n.left
}

func (n *defaultNavigation) Right() key.Binding {
	return n.right
}

func NewDefaultNavigation() Navigation {
	return &defaultNavigation{
		quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q/ctrl+c", "quit"),
		),
		refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		selectItem: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("up", "move up"),
		),
		down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("down", "move down"),
		),
		left: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("left", "move left"),
		),
		right: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("right", "move right"),
		),
	}
}
