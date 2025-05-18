package ui

import (
	"fmt"
	"os"

	"golang.org/x/term"

	"github.com/76creates/stickers/flexbox"
	"github.com/bschimke95/jara/pkg/app"
	"github.com/charmbracelet/lipgloss"
)

// Layout represents the generic layout structure for the application
// It consists of a header, body, and footer
// The layout is designed to be full-screen and responsive

type Layout struct {
	HeaderStyle lipgloss.Style
	BodyStyle   lipgloss.Style
	FooterStyle lipgloss.Style
	Width       int
	Height      int
	flexBox     *flexbox.FlexBox
	heightRatio [3]int // header, body, footer
}

// NewLayout creates a new layout with default styling
func NewLayout() *Layout {
	return &Layout{
		HeaderStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(app.CanonicalTheme.PrimaryColor)).
			Background(lipgloss.Color(app.CanonicalTheme.HeaderBg)),
		BodyStyle: lipgloss.NewStyle().
			Background(lipgloss.Color(app.CanonicalTheme.BodyBg)),
		FooterStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(app.CanonicalTheme.SecondaryColor)).
			Background(lipgloss.Color(app.CanonicalTheme.FooterBg)),
		Width:       80,              // Default width
		Height:      24,              // Default height
		heightRatio: [3]int{1, 8, 1}, // header:body:footer ratio
	}
}

// SetDimensions sets the layout dimensions based on terminal size
func (l *Layout) SetDimensions() error {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return fmt.Errorf("failed to get terminal dimensions: %w", err)
	}
	l.Width = width
	l.Height = height

	// Initialize flexbox with current dimensions
	l.flexBox = flexbox.New(width, height)

	// Calculate total ratio for height distribution
	totalRatio := l.heightRatio[0] + l.heightRatio[1] + l.heightRatio[2]
	heightPerUnit := height / totalRatio

	// Set styles to use full width
	l.HeaderStyle = l.HeaderStyle.Height(heightPerUnit * l.heightRatio[0])
	l.BodyStyle = l.BodyStyle.Height(heightPerUnit * l.heightRatio[1])
	l.FooterStyle = l.FooterStyle.Height(heightPerUnit * l.heightRatio[2])

	return nil
}

// Render renders the layout with the given content
func (l *Layout) Render(header, body, footer string) string {
	// Ensure dimensions are set and flexbox is initialized
	if l.flexBox == nil {
		if err := l.SetDimensions(); err != nil {
			return fmt.Sprintf("Error setting dimensions: %v", err)
		}
	}

	// Create rows for header, body, and footer
	rows := []*flexbox.Row{
		l.flexBox.NewRow().AddCells(
			flexbox.NewCell(1, l.heightRatio[0]).SetContent(
				l.HeaderStyle.Render(header),
			).SetStyle(l.HeaderStyle),
		),
		l.flexBox.NewRow().AddCells(
			flexbox.NewCell(1, l.heightRatio[1]).SetContent(
				l.BodyStyle.Render(body),
			).SetStyle(l.BodyStyle),
		),
		l.flexBox.NewRow().AddCells(
			flexbox.NewCell(1, l.heightRatio[2]).SetContent(
				l.FooterStyle.Render(footer),
			).SetStyle(l.FooterStyle),
		),
	}

	// Add all rows to the flexbox
	l.flexBox.AddRows(rows)

	// Render the flexbox
	return l.flexBox.Render()
}
