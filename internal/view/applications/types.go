package applications

import (
	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
)

// View is the Bubble Tea model for the applications table view.
type View struct {
	table  table.Model
	keys   ui.KeyMap
	width  int
	height int
	status *model.FullStatus
}
