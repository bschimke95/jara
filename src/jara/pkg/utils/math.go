// Package utils provides general utility functions for the application.
package utils

// Min returns the smaller of two integers.
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
