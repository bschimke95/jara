package env

// Theme represents a color theme for the application
// It defines all the necessary colors for different UI elements
type Theme struct {
	// Primary colors
	PrimaryColor   string
	SecondaryColor string

	// Text colors
	PrimaryText   string
	SecondaryText string
	MutedText     string

	// Border colors
	HeaderBorder string
	HeaderText   string // Text color for header
	BodyBorder   string
	FooterBorder string

	SelectedText string
	SelectedBg   string

	// Background colors
	HeaderBg string
	BodyBg   string
	FooterBg string
}

// CanonicalTheme represents the Canonical brand color theme
var CanonicalTheme = Theme{
	// Primary colors
	PrimaryColor:   "208", // Canonical orange
	SecondaryColor: "93",  // Canonical purple

	// Text colors
	PrimaryText:   "255", // White text
	SecondaryText: "245", // Light gray text
	MutedText:     "240", // Gray text

	// Border colors
	HeaderBorder: "240", // Darker gray for header border
	HeaderText:   "255", // White text for header
	BodyBorder:   "242", // Darker gray for body border
	FooterBorder: "245", // Light gray for header/footer borders

	SelectedText: "255",     // Selected text color
	SelectedBg:   "#7D56F4", // Selected background color

	// Background colors
	HeaderBg: "235", // Light background for header
	BodyBg:   "236", // Slightly darker background for body
	FooterBg: "235", // Light background for footer
}
