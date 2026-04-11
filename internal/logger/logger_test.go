package logger

import (
	"log/slog"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"unknown", slog.LevelInfo},
		{"", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLevel(tt.input)
			if got != tt.want {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSetupJSON(t *testing.T) {
	l := Setup("info", "json")
	if l == nil {
		t.Fatal("Setup returned nil")
	}
}

func TestSetupText(t *testing.T) {
	l := Setup("debug", "text")
	if l == nil {
		t.Fatal("Setup returned nil")
	}
}

func TestSetupSetsDefault(t *testing.T) {
	l := Setup("info", "json")
	def := slog.Default()
	if def.Handler() != l.Handler() {
		t.Error("Setup did not set global default logger")
	}
}
