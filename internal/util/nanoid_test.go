package util

import (
	"regexp"
	"testing"
)

func TestNewIDLength(t *testing.T) {
	id := NewID()
	if len(id) != DefaultIDLength {
		t.Errorf("NewID() length = %d, want %d", len(id), DefaultIDLength)
	}
}

func TestNewShortIDLength(t *testing.T) {
	id := NewShortID()
	if len(id) != ShortIDLength {
		t.Errorf("NewShortID() length = %d, want %d", len(id), ShortIDLength)
	}
}

func TestNewIDURLSafe(t *testing.T) {
	// Nanoid default alphabet: A-Za-z0-9_-
	re := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	for i := 0; i < 100; i++ {
		id := NewID()
		if !re.MatchString(id) {
			t.Errorf("NewID() = %q is not URL-safe", id)
		}
	}
}

func TestNewIDUniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := NewID()
		if ids[id] {
			t.Fatalf("duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestNewShortIDUniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := NewShortID()
		if ids[id] {
			t.Fatalf("duplicate short ID generated: %s", id)
		}
		ids[id] = true
	}
}
