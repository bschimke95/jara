package controllers

import (
	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
)

// UpdatedMsg is sent when fresh controller data arrives.
type UpdatedMsg struct {
	Controllers []model.Controller
}

// View is the Bubble Tea model for the controllers table view.
type View struct {
	table       table.Model
	keys        ui.KeyMap
	width       int
	height      int
	controllers []model.Controller
}
