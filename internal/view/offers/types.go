package offers

import (
	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/color"
	"github.com/bschimke95/jara/internal/model"
	"github.com/bschimke95/jara/internal/ui"
)

// FetchOffersMsg signals the app to fetch offers from the API.
type FetchOffersMsg struct{}

// OffersDataMsg carries the fetched offers data back to the view.
type OffersDataMsg struct {
	Offers []model.Offer
	Err    error
}

// View is the Bubble Tea model for the offers table view.
type View struct {
	table     table.Model
	keys      ui.KeyMap
	styles    *color.Styles
	width     int
	height    int
	err       error
	hasData   bool
	offers    []model.Offer
	filterStr string
}
