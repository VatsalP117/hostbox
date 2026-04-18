package hostnames

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/VatsalP117/hostbox/internal/util"
)

const (
	// MaxDNSLabelLength is the RFC 1035 limit for a single DNS label.
	MaxDNSLabelLength = 63
	// PreviewSuffixLength keeps preview URL suffixes short and readable.
	PreviewSuffixLength = 8
	// MaxProjectSlugLength leaves room for "-<preview-suffix>" within one DNS label.
	MaxProjectSlugLength = MaxDNSLabelLength - PreviewSuffixLength - 1

	truncationHashLength = 6
)

// NormalizeProjectSlug returns the canonical DNS-safe project label.
func NormalizeProjectSlug(raw string) string {
	return truncateLabel(util.Slugify(raw), MaxProjectSlugLength)
}

// ProductionHost returns the production hostname for a project.
func ProductionHost(projectSlug, platformDomain string) string {
	return fmt.Sprintf("%s.%s", NormalizeProjectSlug(projectSlug), normalizeDomain(platformDomain))
}

// PreviewHost returns the preview hostname for a deployment.
func PreviewHost(projectSlug, deploymentID, platformDomain string) string {
	label := compoundLabel(NormalizeProjectSlug(projectSlug), previewSuffix(deploymentID))
	return fmt.Sprintf("%s.%s", label, normalizeDomain(platformDomain))
}

// BranchHost returns the branch-stable hostname for a branch deployment.
func BranchHost(projectSlug, branchName, platformDomain string) string {
	label := compoundLabel(NormalizeProjectSlug(projectSlug), util.Slugify(branchName))
	return fmt.Sprintf("%s.%s", label, normalizeDomain(platformDomain))
}

// ReservedProjectLabel returns the reserved platform subdomain label, if the dashboard
// is configured as a direct child of the platform domain (for example hostbox.example.com).
func ReservedProjectLabel(platformDomain, dashboardDomain string) (string, bool) {
	platformDomain = normalizeDomain(platformDomain)
	dashboardDomain = normalizeDomain(dashboardDomain)

	suffix := "." + platformDomain
	if !strings.HasSuffix(dashboardDomain, suffix) {
		return "", false
	}

	prefix := strings.TrimSuffix(dashboardDomain, suffix)
	prefix = strings.TrimSuffix(prefix, ".")
	if prefix == "" || strings.Contains(prefix, ".") {
		return "", false
	}

	return NormalizeProjectSlug(prefix), true
}

// CollidesWithDashboard reports whether the project slug would claim the dashboard host.
func CollidesWithDashboard(projectSlug, platformDomain, dashboardDomain string) bool {
	reserved, ok := ReservedProjectLabel(platformDomain, dashboardDomain)
	if !ok {
		return false
	}
	return NormalizeProjectSlug(projectSlug) == reserved
}

func compoundLabel(head, tail string) string {
	head = truncateLabel(util.Slugify(head), MaxProjectSlugLength)
	tail = util.Slugify(tail)

	if tail == "" {
		return truncateLabel(head, MaxDNSLabelLength)
	}

	tail = truncateLabel(tail, MaxDNSLabelLength-len(head)-1)
	return head + "-" + tail
}

func previewSuffix(value string) string {
	slug := util.Slugify(value)
	if slug == "" {
		return shortHash(value, PreviewSuffixLength)
	}
	slug = truncateLabel(slug, PreviewSuffixLength)
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return shortHash(value, PreviewSuffixLength)
	}
	return slug
}

func truncateLabel(label string, maxLen int) string {
	if maxLen < 1 {
		return "x"
	}
	if len(label) <= maxLen {
		return label
	}
	if maxLen <= truncationHashLength+1 {
		return label[:maxLen]
	}

	prefixLen := maxLen - truncationHashLength - 1
	return fmt.Sprintf("%s-%s", label[:prefixLen], shortHash(label, truncationHashLength))
}

func shortHash(value string, length int) string {
	sum := sha1.Sum([]byte(value))
	encoded := hex.EncodeToString(sum[:])
	if length > len(encoded) {
		length = len(encoded)
	}
	return encoded[:length]
}

func normalizeDomain(domain string) string {
	return strings.Trim(strings.ToLower(strings.TrimSpace(domain)), ".")
}
