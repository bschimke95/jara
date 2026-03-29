package secrets

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"

	"github.com/bschimke95/jara/internal/model"
)

// Columns defines the columns for the secrets table.
func Columns() []table.Column {
	return []table.Column{
		{Title: "URI", Width: 30},
		{Title: "LABEL", Width: 16},
		{Title: "OWNER", Width: 24},
		{Title: "ROTATION", Width: 12},
		{Title: "REV", Width: 5},
		{Title: "UPDATED", Width: 20},
	}
}

// Rows converts a slice of secrets to table rows.
func Rows(secrets []model.Secret) []table.Row {
	rows := make([]table.Row, 0, len(secrets))
	for _, s := range secrets {
		rows = append(rows, table.Row{
			s.URI,
			s.Label,
			formatOwner(s.Owner),
			s.RotatePolicy,
			fmt.Sprintf("%d", s.Revision),
			s.UpdateTime.Format("2006-01-02 15:04:05"),
		})
	}
	return rows
}

// RowsForApp returns secret rows where the owner matches the given application name.
func RowsForApp(secrets []model.Secret, appName string) []table.Row {
	var rows []table.Row
	ownerSuffix := "application-" + appName
	for _, s := range secrets {
		if s.Owner == ownerSuffix {
			rows = append(rows, table.Row{
				s.URI,
				s.Label,
				formatOwner(s.Owner),
				s.RotatePolicy,
				fmt.Sprintf("%d", s.Revision),
				s.UpdateTime.Format("2006-01-02 15:04:05"),
			})
		}
	}
	return rows
}

// formatOwner strips the "application-" or "unit-" prefix for display,
// falling back to the raw string if no known prefix is found.
func formatOwner(owner string) string {
	for _, prefix := range []string{"application-", "unit-", "model-"} {
		if after, ok := strings.CutPrefix(owner, prefix); ok {
			return after
		}
	}
	return owner
}
