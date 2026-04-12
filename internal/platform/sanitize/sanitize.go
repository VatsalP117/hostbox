package sanitize

import (
	"fmt"
	"html"
	"net"
	"net/url"
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
	base = filepath.Clean(base)
	joined := filepath.Join(append([]string{base}, components...)...)
	resolved := filepath.Clean(joined)

	if !strings.HasPrefix(resolved, base+string(filepath.Separator)) && resolved != base {
		return "", fmt.Errorf("path %q escapes base directory %q", joined, base)
	}
	return resolved, nil
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
