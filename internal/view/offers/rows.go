package offers

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/model"
)

// Columns defines the columns for the offers table.
func Columns() []table.Column {
	return []table.Column{
		{Title: "Offer", Width: 22},
		{Title: "Application", Width: 18},
		{Title: "URL", Width: 36},
		{Title: "Endpoints", Width: 20},
		{Title: "Connections", Width: 14},
	}
}

// Rows converts a slice of offers to table rows.
func Rows(offers []model.Offer) []table.Row {
	rows := make([]table.Row, 0, len(offers))
	for _, o := range offers {
		conns := fmt.Sprintf("%d/%d", o.ActiveConnCount, o.TotalConnCount)
		rows = append(rows, table.Row{
			o.Name,
			o.ApplicationName,
			o.OfferURL,
			strings.Join(o.Endpoints, ", "),
			conns,
		})
	}
	return rows
}
