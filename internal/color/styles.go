package color

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"
)

// Styles holds all pre-built styles and semantic colors used throughout the
// TUI. It replaces the former package-level mutable color variables with a
// single immutable value that is constructed once and threaded through the
// application.
type Styles struct {
	// ── Semantic colors (used where raw color.Color is needed) ──────────

	// LogoColor is the bright accent for the logo / branding.
	LogoColor color.Color

	// Primary is the dominant accent (headers, active crumbs, cursors).
	Primary color.Color

	// Secondary is muted informational text.
	Secondary color.Color

	// Title is the default text color (light gray).
	Title color.Color

	// Subtle is used for borders, separators, tree enumerators.
	Subtle color.Color

	// Highlight is the selected-row background.
	Highlight color.Color

	// Muted is for less-important text.
	Muted color.Color

	// HintKeyColor is the raw color for key hint brackets.
	HintKeyColor color.Color

	// CrumbFgColor is the raw foreground for crumb pills.
	CrumbFgColor color.Color

	// CrumbBgColor is the raw background for active crumb pills.
	CrumbBgColor color.Color

	// CrumbBgAltColor is the background for ancestor crumb pills.
	CrumbBgAltColor color.Color

	// SearchHighlightFgColor is the foreground for search-match highlighting.
	SearchHighlightFgColor color.Color

	// SearchHighlightBgColor is the background for search-match highlighting.
	SearchHighlightBgColor color.Color

	// ErrorColor is the raw error color.
	ErrorColor color.Color

	// CheckGreenColor is the raw color for positive check marks.
	CheckGreenColor color.Color

	// CheckRedColor is the raw color for negative check marks.
	CheckRedColor color.Color

	// InfoLabelColor is the raw color for info labels.
	InfoLabelColor color.Color

	// InfoValueColor is the raw color for info values.
	InfoValueColor color.Color

	// HintDescColor is the raw color for hint descriptions.
	HintDescColor color.Color

	// BorderColor is the raw color for borders.
	BorderColor color.Color

	// BorderTitleColor is the raw color for border titles.
	BorderTitleColor color.Color

	// statusColors maps Juju status strings to colors.
	statusColors map[string]color.Color

	// ── Pre-built styles ───────────────────────────────────────────────

	// Logo is the style for the ASCII logo.
	Logo lipgloss.Style

	// Header is bold primary-colored text for table headers.
	Header lipgloss.Style

	// SelectedRow is the full selected-row style (fg + bg + bold).
	SelectedRow lipgloss.Style

	// SelectedRowBgOnly sets only a background highlight (preserves cell fg).
	SelectedRowBgOnly lipgloss.Style

	// Cell is the default table cell style.
	Cell lipgloss.Style

	// UnfocusedHeader is the dimmed header for inactive panes.
	UnfocusedHeader lipgloss.Style

	// UnfocusedSelected is the selected row in an unfocused pane (no highlight).
	UnfocusedSelected lipgloss.Style

	// BorderStyle is the style for box border characters.
	BorderStyle lipgloss.Style

	// BorderTitleStyle is the bold style for titles embedded in borders.
	BorderTitleStyle lipgloss.Style

	// InfoLabel is for dim labels ("Controller:", "Model:").
	InfoLabel lipgloss.Style

	// InfoValue is for values next to info labels.
	InfoValue lipgloss.Style

	// HintKey is bold colored key hints ("<q>", "<enter>").
	HintKey lipgloss.Style

	// HintDesc is the muted description next to key hints.
	HintDesc lipgloss.Style

	// HintSep is the separator between key hints.
	HintSep lipgloss.Style

	// CrumbActive is the active (current) breadcrumb pill.
	CrumbActive lipgloss.Style

	// CrumbAncestor is the ancestor breadcrumb pill.
	CrumbAncestor lipgloss.Style

	// CrumbSep is the separator between breadcrumb pills.
	CrumbSep lipgloss.Style

	// ErrorStyle is for error messages.
	ErrorStyle lipgloss.Style

	// MutedText is for less-important text.
	MutedText lipgloss.Style

	// TitleText is for default-colored text.
	TitleText lipgloss.Style

	// PrimaryText is bold primary-colored text (used for various titles).
	PrimaryText lipgloss.Style

	// SecondaryText is for secondary informational text.
	SecondaryText lipgloss.Style

	// SubtleText is for very low-prominence text (borders, separators).
	SubtleText lipgloss.Style

	// Cursor is the cursor block style (primary foreground).
	Cursor lipgloss.Style

	// SearchHighlight is the style for search-match highlighting.
	SearchHighlight lipgloss.Style

	// CheckGreen is the style for positive check marks.
	CheckGreen lipgloss.Style

	// CheckRed is the style for negative/unchecked marks.
	CheckRed lipgloss.Style

	// Pending is italic muted text for placeholder/pending rows.
	Pending lipgloss.Style
}

// StatusColor returns the appropriate color for a Juju status string.
func (s *Styles) StatusColor(status string) color.Color {
	if c, ok := s.statusColors[status]; ok {
		return c
	}
	if c, ok := s.statusColors[""]; ok {
		return c
	}
	return lipgloss.Color("#c0c0c0")
}

// StatusStyle returns a lipgloss style with the foreground set to the status color.
func (s *Styles) StatusStyle(status string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(s.StatusColor(status))
}

// StatusText renders a Juju status string with its semantic foreground color
// while preserving any parent-applied background style.
func (s *Styles) StatusText(status string) string {
	return ForegroundText(s.StatusColor(status), status)
}

// DefaultStyles returns the compiled-in default styles (Atom One Dark).
func DefaultStyles() *Styles {
	return newStyles(defaultPalette())
}

// palette holds the raw color values from which styles are derived.
type palette struct {
	logoColor         color.Color
	primary           color.Color
	secondary         color.Color
	title             color.Color
	subtle            color.Color
	highlight         color.Color
	muted             color.Color
	hintKey           color.Color
	hintDesc          color.Color
	crumbFg           color.Color
	crumbBg           color.Color
	border            color.Color
	borderTitle       color.Color
	infoLabel         color.Color
	infoValue         color.Color
	errorColor        color.Color
	searchHighlightFg color.Color
	searchHighlightBg color.Color
	crumbBgAlt        color.Color
	checkGreen        color.Color
	checkRed          color.Color
	statusColors      map[string]color.Color
}

// defaultPalette returns the Atom One Dark color palette.
func defaultPalette() palette {
	return palette{
		logoColor:         lipgloss.Color("#61afef"),
		primary:           lipgloss.Color("#61afef"),
		secondary:         lipgloss.Color("#5c6370"),
		title:             lipgloss.Color("#abb2bf"),
		subtle:            lipgloss.Color("#4b5263"),
		highlight:         lipgloss.Color("#3e4451"),
		muted:             lipgloss.Color("#5c6370"),
		hintKey:           lipgloss.Color("#e5c07b"),
		hintDesc:          lipgloss.Color("#5c6370"),
		crumbFg:           lipgloss.Color("#282c34"),
		crumbBg:           lipgloss.Color("#61afef"),
		border:            lipgloss.Color("#4b5263"),
		borderTitle:       lipgloss.Color("#61afef"),
		infoLabel:         lipgloss.Color("#5c6370"),
		infoValue:         lipgloss.Color("#abb2bf"),
		errorColor:        lipgloss.Color("#e06c75"),
		searchHighlightFg: lipgloss.Color("#282c34"),
		searchHighlightBg: lipgloss.Color("#e5c07b"),
		crumbBgAlt:        lipgloss.Color("#3e4451"),
		checkGreen:        lipgloss.Color("#98c379"),
		checkRed:          lipgloss.Color("#e06c75"),
		statusColors: map[string]color.Color{
			"active":      lipgloss.Color("#00ff00"),
			"idle":        lipgloss.Color("#00ff00"),
			"running":     lipgloss.Color("#00ff00"),
			"started":     lipgloss.Color("#00ff00"),
			"blocked":     lipgloss.Color("#ff5555"),
			"error":       lipgloss.Color("#ff5555"),
			"lost":        lipgloss.Color("#ff5555"),
			"down":        lipgloss.Color("#ff5555"),
			"waiting":     lipgloss.Color("#ffff00"),
			"allocating":  lipgloss.Color("#ffff00"),
			"pending":     lipgloss.Color("#ffff00"),
			"maintenance": lipgloss.Color("#00bfff"),
			"executing":   lipgloss.Color("#00bfff"),
			"terminated":  lipgloss.Color("#808080"),
			"unknown":     lipgloss.Color("#808080"),
			"stopped":     lipgloss.Color("#ff4500"),
		},
	}
}

// newStyles builds a complete Styles value from a palette.
func newStyles(p palette) *Styles {
	s := &Styles{
		// Raw colors
		LogoColor:              p.logoColor,
		Primary:                p.primary,
		Secondary:              p.secondary,
		Title:                  p.title,
		Subtle:                 p.subtle,
		Highlight:              p.highlight,
		Muted:                  p.muted,
		HintKeyColor:           p.hintKey,
		HintDescColor:          p.hintDesc,
		CrumbFgColor:           p.crumbFg,
		CrumbBgColor:           p.crumbBg,
		CrumbBgAltColor:        p.crumbBgAlt,
		SearchHighlightFgColor: p.searchHighlightFg,
		SearchHighlightBgColor: p.searchHighlightBg,
		ErrorColor:             p.errorColor,
		CheckGreenColor:        p.checkGreen,
		CheckRedColor:          p.checkRed,
		InfoLabelColor:         p.infoLabel,
		InfoValueColor:         p.infoValue,
		BorderColor:            p.border,
		BorderTitleColor:       p.borderTitle,
		statusColors:           p.statusColors,
	}

	s.RebuildStyles()
	return s
}

// SetStatusColor sets or overrides the color for a specific status key.
func (s *Styles) SetStatusColor(key string, c color.Color) {
	if s.statusColors == nil {
		s.statusColors = make(map[string]color.Color)
	}
	s.statusColors[key] = c
}

// RebuildStyles reconstructs all pre-built lipgloss.Style values from the
// current raw color fields. Call this after mutating any color on the struct
// (e.g. after applying config overrides).
func (s *Styles) RebuildStyles() {
	s.Logo = lipgloss.NewStyle().Foreground(s.LogoColor).Bold(true)

	s.Header = lipgloss.NewStyle().Bold(true).Foreground(s.Primary).Padding(0, 1)
	s.SelectedRow = lipgloss.NewStyle().
		Foreground(s.CrumbFgColor).
		Background(s.Highlight).
		Bold(true)
	s.SelectedRowBgOnly = lipgloss.NewStyle().
		Background(s.Highlight).
		Bold(true)
	s.Cell = lipgloss.NewStyle().Padding(0, 1)

	s.UnfocusedHeader = lipgloss.NewStyle().Bold(true).Foreground(s.Muted).Padding(0, 1)
	s.UnfocusedSelected = lipgloss.NewStyle().Foreground(s.Title)

	s.BorderStyle = lipgloss.NewStyle().Foreground(s.BorderColor)
	s.BorderTitleStyle = lipgloss.NewStyle().Foreground(s.BorderTitleColor).Bold(true)

	s.InfoLabel = lipgloss.NewStyle().Foreground(s.InfoLabelColor)
	s.InfoValue = lipgloss.NewStyle().Foreground(s.InfoValueColor)

	s.HintKey = lipgloss.NewStyle().Foreground(s.HintKeyColor).Bold(true)
	s.HintDesc = lipgloss.NewStyle().Foreground(s.HintDescColor)
	s.HintSep = lipgloss.NewStyle().Foreground(s.Subtle)

	s.CrumbActive = lipgloss.NewStyle().
		Foreground(s.CrumbFgColor).
		Background(s.CrumbBgColor).
		Bold(true).
		Padding(0, 1)
	s.CrumbAncestor = lipgloss.NewStyle().
		Foreground(s.CrumbFgColor).
		Background(s.CrumbBgAltColor).
		Padding(0, 1)
	s.CrumbSep = lipgloss.NewStyle().Foreground(s.Subtle)

	s.ErrorStyle = lipgloss.NewStyle().Foreground(s.ErrorColor)
	s.MutedText = lipgloss.NewStyle().Foreground(s.Muted)
	s.TitleText = lipgloss.NewStyle().Foreground(s.Title)
	s.PrimaryText = lipgloss.NewStyle().Foreground(s.Primary).Bold(true)
	s.SecondaryText = lipgloss.NewStyle().Foreground(s.Secondary)
	s.SubtleText = lipgloss.NewStyle().Foreground(s.Subtle)
	s.Cursor = lipgloss.NewStyle().Foreground(s.Primary)

	s.SearchHighlight = lipgloss.NewStyle().
		Foreground(s.SearchHighlightFgColor).
		Background(s.SearchHighlightBgColor)

	s.CheckGreen = lipgloss.NewStyle().Foreground(s.CheckGreenColor).Bold(true)
	s.CheckRed = lipgloss.NewStyle().Foreground(s.CheckRedColor)

	s.Pending = lipgloss.NewStyle().Foreground(s.Muted).Italic(true)
}

// ForegroundText renders text with only a foreground color sequence and a
// foreground-only reset. This preserves any outer background highlight.
func ForegroundText(c color.Color, text string) string {
	if c == nil || text == "" {
		return text
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[39m", r>>8, g>>8, b>>8, text)
}
