package config

import (
	"charm.land/bubbles/v2/key"

	"github.com/bschimke95/jara/internal/ui"
)

// ResolveKeyMap builds a KeyMap by applying KeysConfig overrides on top of the
// compiled defaults. Any binding left nil in the config retains its default.
func ResolveKeyMap(keys KeysConfig) ui.KeyMap {
	km := ui.DefaultKeyMap()

	overrideBinding(&km.Quit, keys.Quit, "quit")
	overrideBinding(&km.Help, keys.Help, "help")
	overrideBinding(&km.Back, keys.Back, "back")
	overrideBinding(&km.Enter, keys.Enter, "select")
	overrideBinding(&km.Command, keys.Command, "command")
	overrideBinding(&km.Filter, keys.Filter, "filter")
	overrideBinding(&km.Up, keys.Up, "up")
	overrideBinding(&km.Down, keys.Down, "down")
	overrideBinding(&km.PageUp, keys.PageUp, "page up")
	overrideBinding(&km.PageDown, keys.PageDown, "page down")
	overrideBinding(&km.Top, keys.Top, "top")
	overrideBinding(&km.Bottom, keys.Bottom, "bottom")
	overrideBinding(&km.CancelInput, keys.CancelInput, "cancel")
	overrideBinding(&km.Tab, keys.Tab, "switch pane")
	overrideBinding(&km.ScaleUp, keys.ScaleUp, "scale up")
	overrideBinding(&km.ScaleDown, keys.ScaleDown, "scale down")
	overrideBinding(&km.Deploy, keys.Deploy, "deploy")
	overrideBinding(&km.Relate, keys.Relate, "relate")
	overrideBinding(&km.DeleteRelation, keys.DeleteRelation, "delete relation")
	overrideBinding(&km.LogsJump, keys.LogsJump, "logs")
	overrideBinding(&km.LogsView, keys.LogsView, "logs")
	overrideBinding(&km.ClearFilter, keys.ClearFilter, "clear filter")
	overrideBinding(&km.SearchOpen, keys.SearchOpen, "search")
	overrideBinding(&km.SearchNext, keys.SearchNext, "next match")
	overrideBinding(&km.SearchPrev, keys.SearchPrev, "prev match")
	overrideBinding(&km.FilterOpen, keys.FilterOpen, "filter")
	overrideBinding(&km.UnitsNav, keys.UnitsNav, "units")
	overrideBinding(&km.ApplicationsNav, keys.ApplicationsNav, "applications")
	overrideBinding(&km.RelationsNav, keys.RelationsNav, "relations")
	overrideBinding(&km.SecretsNav, keys.SecretsNav, "secrets")
	overrideBinding(&km.MachinesNav, keys.MachinesNav, "machines")
	overrideBinding(&km.OffersNav, keys.OffersNav, "offers")
	overrideBinding(&km.StorageNav, keys.StorageNav, "storage")
	overrideBinding(&km.Decode, keys.Decode, "decode")
	overrideBinding(&km.Yank, keys.Yank, "copy")
	overrideBinding(&km.ApplyFilter, keys.ApplyFilter, "apply")
	overrideBinding(&km.Right, keys.Right, "right")
	overrideBinding(&km.Left, keys.Left, "left")
	overrideBinding(&km.RunAction, keys.RunAction, "run action")
	overrideBinding(&km.ConfigNav, keys.ConfigNav, "config")
	overrideBinding(&km.EntitySwitch, keys.EntitySwitch, "switch entity")
	overrideBinding(&km.NewModel, keys.NewModel, "new model")
	overrideBinding(&km.RemoveModel, keys.RemoveModel, "remove model")

	return km
}

func overrideBinding(target *key.Binding, override *KeyBindingConfig, helpDesc string) {
	if override == nil || len(override.Keys) == 0 {
		return
	}
	// Build the help key from the first key in the list.
	helpKey := override.Keys[0]
	*target = key.NewBinding(
		key.WithKeys(override.Keys...),
		key.WithHelp(helpKey, helpDesc),
	)
}
