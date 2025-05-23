package ui

import (
	"fmt"
	"os"

	"golang.org/x/term"

	"github.com/76creates/stickers/flexbox"
	"github.com/bschimke95/jara/pkg/env"
	"github.com/charmbracelet/lipgloss"
)

// Layout represents the generic layout structure for the application
// It consists of a header and body
// The layout is designed to be full-screen and responsive
type Layout struct {
	Header      string
	HeaderStyle lipgloss.Style
	BodyStyle   lipgloss.Style
	Width       int
	Height      int
	flexBox     *flexbox.FlexBox
	heightRatio [2]int // header, body
	headerInfo  HeaderInfo // Store HeaderInfo for later rendering
}

func WithHeader(header HeaderInfo) func(*Layout) {
	return func(l *Layout) {
		// We'll set the header after dimensions are set to ensure correct width
		l.headerInfo = header
	}
}

// NewLayout creates a new layout with default styling
func NewLayout(options ...func(*Layout)) *Layout {
	layout := &Layout{
		HeaderStyle: lipgloss.NewStyle().
			Background(lipgloss.Color(env.CanonicalTheme.HeaderBg)),
		BodyStyle: lipgloss.NewStyle().
			Background(lipgloss.Color(env.CanonicalTheme.BodyBg)).
			Width(80). // Will be resized later
			Height(24), // Will be resized later
		Width:       80, // Default width, will be updated
		Height:      24, // Default height, will be updated
		heightRatio: [2]int{6, 94}, // Much smaller header to maximize content space
	}

	for _, option := range options {
		option(layout)
	}

	return layout
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
	totalRatio := l.heightRatio[0] + l.heightRatio[1]
	heightPerUnit := height / totalRatio

	// Set header height and width
	headerHeight := heightPerUnit * l.heightRatio[0]
	l.HeaderStyle = l.HeaderStyle.Width(width).Height(headerHeight)

	// Set body style to fill remaining space
	bodyHeight := heightPerUnit * l.heightRatio[1]
	l.BodyStyle = l.BodyStyle.Width(width).Height(bodyHeight)

	// If we have header info set, render it with the correct width
	if l.headerInfo.KeyHints != nil {
		l.Header = Header(width, l.headerInfo)
	}

	return nil
}

// Render renders the layout with the given content
func (l *Layout) Render(body string) string {
	// Ensure dimensions are set and flexbox is initialized
	if l.flexBox == nil {
		if err := l.SetDimensions(); err != nil {
			return fmt.Sprintf("Error setting dimensions: %v", err)
		}
	}

	// Create cells that fill the entire width (1) and appropriate height ratio
	rows := []*flexbox.Row{
		l.flexBox.NewRow().AddCells(
			flexbox.NewCell(1, l.heightRatio[0]).
				SetContent(l.HeaderStyle.Render(l.Header)).
				SetStyle(l.HeaderStyle),
		),
		l.flexBox.NewRow().AddCells(
			flexbox.NewCell(1, l.heightRatio[1]).
				SetContent(l.BodyStyle.Render(body)).
				SetStyle(l.BodyStyle),
		),
	}

	l.flexBox.AddRows(rows)
	return l.flexBox.Render()
}
