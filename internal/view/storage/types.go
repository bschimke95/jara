package storage

import (
	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
)

// FetchStorageMsg requests that storage data be fetched from the API.
type FetchStorageMsg struct{}

// StorageDataMsg carries the storage data response.
type StorageDataMsg struct {
	Instances []model.StorageInstance
	Err       error
}

// View is the Bubble Tea model for the storage table view.
type View struct {
	table  table.Model
	keys   ui.KeyMap
	styles *color.Styles
	width  int
	height int
}
