package sanitize

import (
	"fmt"
	"html"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// SanitizeLogLine escapes HTML-sensitive characters in build log output
// to prevent XSS when displayed in SSE streams or web UIs.
func SanitizeLogLine(line string) string {
	return html.EscapeString(line)
}

// SafeJoinPath joins path components and ensures the result is under the base directory.
// Returns an error if the resolved path escapes the base directory.
func SafeJoinPath(base string, components ...string) (string, error) {
	base, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("resolve base directory: %w", err)
	}
	base = filepath.Clean(base)
	joined := filepath.Join(append([]string{base}, components...)...)
	resolved := filepath.Clean(joined)

	if !pathWithinBase(base, resolved) {
		return "", fmt.Errorf("path %q escapes base directory %q", joined, base)
	}

	canonicalBase, err := canonicalizeExistingPath(base)
	if err != nil {
		return "", fmt.Errorf("resolve base directory: %w", err)
	}
	canonicalResolved, err := canonicalizeExistingPath(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve path %q: %w", resolved, err)
	}
	if !pathWithinBase(canonicalBase, canonicalResolved) {
		return "", fmt.Errorf("path %q escapes base directory %q through a symbolic link", joined, base)
	}
	return resolved, nil
}

// SafeRelativePath normalizes a user-configured relative path and rejects
// absolute paths and traversal. A leading slash is accepted for project root
// compatibility and is interpreted relative to base ("/apps/web" => "apps/web").
func SafeRelativePath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "/" || raw == "." {
		return ".", nil
	}

	rel := strings.TrimLeft(raw, `/\\`)
	clean := filepath.Clean(rel)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || filepath.IsAbs(clean) {
		return "", fmt.Errorf("path %q must stay within its configured root", raw)
	}
	return clean, nil
}

func pathWithinBase(base, candidate string) bool {
	rel, err := filepath.Rel(base, candidate)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// canonicalizeExistingPath resolves symlinks in every existing component while
// retaining a possibly non-existent suffix (needed for build output directories).
func canonicalizeExistingPath(path string) (string, error) {
	path = filepath.Clean(path)
	existing := path
	var suffix []string
	for {
		_, err := os.Lstat(existing)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return "", err
		}
		suffix = append(suffix, filepath.Base(existing))
		existing = parent
	}

	resolved, err := filepath.EvalSymlinks(existing)
	if err != nil {
		return "", err
	}
	for i := len(suffix) - 1; i >= 0; i-- {
		resolved = filepath.Join(resolved, suffix[i])
	}
	return filepath.Clean(resolved), nil
}

// ValidateWebhookURL ensures URL is HTTPS and not targeting internal/private IPs.
func ValidateWebhookURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme != "https" {
		return fmt.Errorf("webhook URL must use HTTPS, got %q", u.Scheme)
	}

	if u.Host == "" {
		return fmt.Errorf("webhook URL must have a host")
	}

	hostname := u.Hostname()

	// Block dangerous schemes disguised in host
	if strings.Contains(hostname, "javascript") || strings.Contains(hostname, "data:") {
		return fmt.Errorf("invalid hostname %q", hostname)
	}

	// Resolve hostname to check for private IPs
	ips, err := net.LookupIP(hostname)
	if err != nil {
		// If DNS lookup fails, allow it (might be a valid hostname not resolvable from this host)
		return nil
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("webhook URL must not target private IP (resolved %s to %s)", hostname, ip)
		}
	}

	return nil
}

func isPrivateIP(ip net.IP) bool {
	privateRanges := []struct {
		start net.IP
		end   net.IP
	}{
		{net.ParseIP("10.0.0.0"), net.ParseIP("10.255.255.255")},
		{net.ParseIP("172.16.0.0"), net.ParseIP("172.31.255.255")},
		{net.ParseIP("192.168.0.0"), net.ParseIP("192.168.255.255")},
		{net.ParseIP("127.0.0.0"), net.ParseIP("127.255.255.255")},
		{net.ParseIP("169.254.0.0"), net.ParseIP("169.254.255.255")},
	}

	ip4 := ip.To4()
	if ip4 == nil {
		// Check IPv6 loopback
		return ip.Equal(net.IPv6loopback)
	}

	for _, r := range privateRanges {
		if bytesInRange(ip4, r.start.To4(), r.end.To4()) {
			return true
		}
	}
	return false
}

func bytesInRange(ip, start, end net.IP) bool {
	for i := 0; i < len(ip); i++ {
		if ip[i] < start[i] {
			return false
		}
		if ip[i] > end[i] {
			return false
		}
	}
	return true
}
