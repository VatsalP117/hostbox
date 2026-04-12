package sanitize

import (
	"testing"
)

func TestSanitizeLogLine(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal log line", "normal log line"},
		{"<script>alert(1)</script>", "&lt;script&gt;alert(1)&lt;/script&gt;"},
		{`user "admin" & <admin>`, `user &#34;admin&#34; &amp; &lt;admin&gt;`},
		{"", ""},
		{"no special chars 123", "no special chars 123"},
		{`<img src=x onerror="alert('xss')">`, `&lt;img src=x onerror=&#34;alert(&#39;xss&#39;)&#34;&gt;`},
	}

	for _, tt := range tests {
		got := SanitizeLogLine(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeLogLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSafeJoinPath(t *testing.T) {
	tests := []struct {
		base       string
		components []string
		wantErr    bool
		wantPath   string
	}{
		{"/app/deployments", []string{"proj-1", "build-abc"}, false, "/app/deployments/proj-1/build-abc"},
		{"/app/deployments", []string{"..", "etc", "passwd"}, true, ""},
		{"/app/deployments", []string{"../../etc/passwd"}, true, ""},
		{"/app/deployments", []string{"proj-1", "..", "..", "secrets"}, true, ""},
		{"/app/deployments", []string{"valid-slug"}, false, "/app/deployments/valid-slug"},
		{"/app/deployments", []string{"a/b/c"}, false, "/app/deployments/a/b/c"},
	}

	for _, tt := range tests {
		got, err := SafeJoinPath(tt.base, tt.components...)
		if (err != nil) != tt.wantErr {
			t.Errorf("SafeJoinPath(%q, %v) error = %v, wantErr %v", tt.base, tt.components, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.wantPath {
			t.Errorf("SafeJoinPath(%q, %v) = %q, want %q", tt.base, tt.components, got, tt.wantPath)
		}
	}
}

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https", "https://hooks.slack.com/services/T00000/B00000/XXXX", false},
		{"http rejected", "http://hooks.slack.com/services/T00000/B00000/XXXX", true},
		{"javascript rejected", "javascript:alert(1)", true},
		{"empty scheme", "://no-scheme.com", true},
		{"no host", "https://", true},
		{"file scheme", "file:///etc/passwd", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWebhookURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip      string
		private bool
	}{
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"127.0.0.1", true},
		{"169.254.169.254", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"203.0.113.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := parseIP(t, tt.ip)
			got := isPrivateIP(ip)
			if got != tt.private {
				t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, got, tt.private)
			}
		})
	}
}

func parseIP(t *testing.T, s string) []byte {
	t.Helper()
	parts := make([]byte, 4)
	n := 0
	val := 0
	for _, c := range s {
		if c == '.' {
			parts[n] = byte(val)
			n++
			val = 0
		} else {
			val = val*10 + int(c-'0')
		}
	}
	parts[n] = byte(val)
	return parts
}
