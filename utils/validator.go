package utils

import (
	"fmt"
	"html"
	"strings"
)

const (
	// DefaultRadiusKm keeps nearby searches focused on the user's immediate area.
	DefaultRadiusKm = 1.5
	MaxRadiusKm     = 20.0
)

// SanitizeString trims whitespace and escapes HTML entities to prevent XSS.
func SanitizeString(s string) string {
	return html.EscapeString(strings.TrimSpace(s))
}

// ValidateCoordinates checks that latitude and longitude are within valid ranges.
func ValidateCoordinates(lat, lng float64) error {
	if lat < -90 || lat > 90 {
		return fmt.Errorf("latitud debe estar entre -90 y 90, recibido: %f", lat)
	}
	if lng < -180 || lng > 180 {
		return fmt.Errorf("longitud debe estar entre -180 y 180, recibido: %f", lng)
	}
	return nil
}

// ValidatePagination normalizes and validates pagination parameters.
// Returns (page, pageSize) with safe defaults.
func ValidatePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

// PaginationOffset calculates the SQL OFFSET from page and pageSize.
func PaginationOffset(page, pageSize int) int {
	return (page - 1) * pageSize
}

// ValidateRadius ensures the search radius is within acceptable bounds (km).
func ValidateRadius(radius float64) float64 {
	if radius <= 0 {
		return DefaultRadiusKm
	}
	if radius > MaxRadiusKm {
		return MaxRadiusKm
	}
	return radius
}
