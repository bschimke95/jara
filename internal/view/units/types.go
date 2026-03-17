package units

import (
	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
)

// View is the Bubble Tea model for the units table view.
type View struct {
	table        table.Model
	keys         ui.KeyMap
	width        int
	height       int
	status       *model.FullStatus
	appName      string
	pendingScale map[string]int // net pending unit delta per app
}
