package overview

import (
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
)

// View is the Bubble Tea model for the tree overview.
type View struct {
	keys   ui.KeyMap
	width  int
	height int
	status *model.FullStatus
}
