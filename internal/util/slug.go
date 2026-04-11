package util

import (
	"regexp"
	"strings"
)

var (
	nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)
	leadingTrailing = regexp.MustCompile(`^-+|-+$`)
)

// Slugify converts a string to a URL-safe slug.
func Slugify(s string) string {
	slug := strings.ToLower(s)
	slug = nonAlphanumeric.ReplaceAllString(slug, "-")
	slug = leadingTrailing.ReplaceAllString(slug, "")
	if slug == "" {
		slug = "project"
	}
	return slug
}
