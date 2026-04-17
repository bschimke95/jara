package machines

import (
	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
)

// View is the Bubble Tea model for the machines table view.
type View struct {
	table     table.Model
	keys      ui.KeyMap
	styles    *color.Styles
	width     int
	height    int
	status    *model.FullStatus
	filterStr string
}
