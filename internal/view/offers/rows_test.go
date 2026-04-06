package offers

import (
	"testing"

	"github.com/bschimke95/jara/internal/model"
)

func TestColumns(t *testing.T) {
	cols := Columns()
	if len(cols) != 5 {
		t.Fatalf("Columns() returned %d columns, want 5", len(cols))
	}

	want := []string{"Offer", "Application", "URL", "Endpoints", "Connections"}
	for i, c := range cols {
		if c.Title != want[i] {
			t.Errorf("Columns()[%d].Title = %q, want %q", i, c.Title, want[i])
		}
		if c.Width <= 0 {
			t.Errorf("Columns()[%d].Width = %d, want > 0", i, c.Width)
		}
	}
}

func TestRows(t *testing.T) {
	tests := []struct {
		name   string
		offers []model.Offer
		want   [][]string // expected row values
	}{
		{
			name:   "empty slice",
			offers: nil,
			want:   nil,
		},
		{
			name: "single offer no endpoints",
			offers: []model.Offer{
				{
					Name:            "my-offer",
					ApplicationName: "mysql",
					OfferURL:        "admin/prod.mysql",
					Endpoints:       nil,
					ActiveConnCount: 0,
					TotalConnCount:  0,
				},
			},
			want: [][]string{
				{"my-offer", "mysql", "admin/prod.mysql", "", "0/0"},
			},
		},
		{
			name: "single offer with endpoints",
			offers: []model.Offer{
				{
					Name:            "db-offer",
					ApplicationName: "postgresql",
					OfferURL:        "admin/staging.postgresql",
					Endpoints:       []string{"db", "db-admin"},
					ActiveConnCount: 3,
					TotalConnCount:  5,
				},
			},
			want: [][]string{
				{"db-offer", "postgresql", "admin/staging.postgresql", "db, db-admin", "3/5"},
			},
		},
		{
			name: "multiple offers",
			offers: []model.Offer{
				{
					Name:            "offer-a",
					ApplicationName: "app-a",
					OfferURL:        "url-a",
					Endpoints:       []string{"ep1"},
					ActiveConnCount: 1,
					TotalConnCount:  2,
				},
				{
					Name:            "offer-b",
					ApplicationName: "app-b",
					OfferURL:        "url-b",
					Endpoints:       []string{"ep1", "ep2", "ep3"},
					ActiveConnCount: 10,
					TotalConnCount:  20,
				},
			},
			want: [][]string{
				{"offer-a", "app-a", "url-a", "ep1", "1/2"},
				{"offer-b", "app-b", "url-b", "ep1, ep2, ep3", "10/20"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := Rows(tt.offers)
			if len(rows) != len(tt.want) {
				t.Fatalf("Rows() returned %d rows, want %d", len(rows), len(tt.want))
			}
			for i, row := range rows {
				if len(row) != 5 {
					t.Fatalf("row[%d] has %d columns, want 5", i, len(row))
				}
				for j, cell := range row {
					if cell != tt.want[i][j] {
						t.Errorf("row[%d][%d] = %q, want %q", i, j, cell, tt.want[i][j])
					}
				}
			}
		})
	}
}
