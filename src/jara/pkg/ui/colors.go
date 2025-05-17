package ui

// Theme represents a color theme for the application
// It defines all the necessary colors for different UI elements
type Theme struct {
	// Primary colors
	PrimaryColor string
	SecondaryColor string
	
	// Text colors
	PrimaryText string
	SecondaryText string
	
	// Border colors
	HeaderBorder string
	BodyBorder string
	FooterBorder string
	
	// Background colors
	HeaderBg string
	BodyBg string
	FooterBg string
}

// CanonicalTheme represents the Canonical brand color theme
var CanonicalTheme = Theme{
	// Primary colors
	PrimaryColor: "208",  // Canonical orange
	SecondaryColor: "93", // Canonical purple
	
	// Text colors
	PrimaryText: "240",  // Dark gray for primary text
	SecondaryText: "245", // Light gray for secondary text
	
	// Border colors
	HeaderBorder: "245", // Light gray for header/footer borders
	BodyBorder: "242",  // Darker gray for body border
	FooterBorder: "245", // Light gray for header/footer borders
	
	// Background colors
	HeaderBg: "235", // Light background for header
	BodyBg: "236",   // Slightly darker background for body
	FooterBg: "235", // Light background for footer
}
