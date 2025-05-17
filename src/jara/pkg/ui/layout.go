package ui

import (
	"fmt"
	"os"

	"golang.org/x/term"

	"github.com/76creates/stickers/flexbox"
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

	// Flex definitions
	flexBox *flexbox.FlexBox
}

// NewLayout creates a new layout with default styling
func NewLayout() *Layout {
	return &Layout{
		HeaderStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(CanonicalTheme.PrimaryColor)).
			Background(lipgloss.Color(CanonicalTheme.HeaderBg)).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(CanonicalTheme.HeaderBorder)),
		BodyStyle: lipgloss.NewStyle().
			Background(lipgloss.Color(CanonicalTheme.BodyBg)).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(CanonicalTheme.BodyBorder)),
		FooterStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(CanonicalTheme.SecondaryColor)).
			Background(lipgloss.Color(CanonicalTheme.FooterBg)).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(CanonicalTheme.FooterBorder)),
		Width:   80, // Default width
		Height:  24, // Default height
		flexBox: flexbox.New(0, 0),
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

	// Update flex containers with current dimensions
	l.flexBox.SetWidth(width)
	l.flexBox.SetHeight(height)

	return nil
}

// Render renders the layout with the given content
func (l *Layout) Render(header, body, footer string) string {
	// Create rows for header, body, and footer
	rows := []*flexbox.Row{
		l.flexBox.NewRow().AddCells(
			flexbox.NewCell(100, 10).SetContent(l.HeaderStyle.Render(header)),
		),
		l.flexBox.NewRow().AddCells(
			flexbox.NewCell(100, 85).SetContent(l.BodyStyle.Render(body)),
		),
		l.flexBox.NewRow().AddCells(
			flexbox.NewCell(100, 5).
				SetContent(lipgloss.JoinHorizontal(
					lipgloss.Right,
					"",
					l.FooterStyle.Render(footer),
				)).
				SetStyle(l.FooterStyle),
		),
	}

	l.flexBox.AddRows(rows)
	return l.flexBox.Render()
}

// DefaultLayout returns a pre-configured layout with standard styling
func DefaultLayout() *Layout {
	layout := NewLayout()
	layout.SetDimensions() // Set to current terminal dimensions
	return layout
}
