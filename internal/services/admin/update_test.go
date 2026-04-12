package admin

import "testing"

func TestSemverGreater(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"1.0.0", "0.9.0", true},
		{"1.1.0", "1.0.0", true},
		{"1.0.1", "1.0.0", true},
		{"2.0.0", "1.9.9", true},
		{"1.0.0", "1.0.0", false},
		{"0.9.0", "1.0.0", false},
		{"1.0.0", "1.0.1", false},
		{"1.10.0", "1.9.0", true},
		{"1", "0.9.0", true},
		{"", "", false},
	}

	for _, tt := range tests {
		got := semverGreater(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("semverGreater(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
