package caddy

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// isSPAFramework returns true if the framework uses client-side routing.
func isSPAFramework(framework string) bool {
	switch framework {
	case "nextjs", "vite", "cra", "svelte", "angular", "vue", "sveltekit":
		return true
	case "astro", "gatsby", "nuxt", "hugo", "html", "static", "":
		return false
	default:
		return true
	}
}

var nonAlphanumericRe = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts a string to a URL-safe slug.
func Slugify(s string) string {
	s = strings.ToLower(s)
	s = norm.NFKD.String(s)
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			return r
		}
		return '-'
	}, s)
	s = nonAlphanumericRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "default"
	}
	return s
}

// groupByProject groups deployments by their ProjectID.
func groupByProject(deployments []ActiveDeployment) map[string][]ActiveDeployment {
	result := make(map[string][]ActiveDeployment)
	for _, d := range deployments {
		result[d.ProjectID] = append(result[d.ProjectID], d)
	}
	return result
}
