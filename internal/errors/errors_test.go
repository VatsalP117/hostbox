package errors

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestAppErrorImplementsError(t *testing.T) {
	var _ error = &AppError{}
}

func TestErrorString(t *testing.T) {
	e := NewNotFound("project")
	got := e.Error()
	want := "NOT_FOUND: project not found"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestErrorStringWithInternal(t *testing.T) {
	inner := fmt.Errorf("disk full")
	e := NewInternal(inner)
	got := e.Error()
	if got == "" {
		t.Fatal("Error() returned empty string")
	}
	if !contains(got, "disk full") {
		t.Errorf("Error() = %q, want to contain 'disk full'", got)
	}
}

func TestUnwrap(t *testing.T) {
	inner := fmt.Errorf("connection refused")
	e := NewInternal(inner)
	if e.Unwrap() != inner {
		t.Errorf("Unwrap() did not return inner error")
	}
}

func TestUnwrapNil(t *testing.T) {
	e := NewNotFound("user")
	if e.Unwrap() != nil {
		t.Errorf("Unwrap() = %v, want nil", e.Unwrap())
	}
}

func TestErrorsAs(t *testing.T) {
	e := NewNotFound("user")
	var appErr *AppError
	if !errors.As(e, &appErr) {
		t.Fatal("errors.As failed for *AppError")
	}
	if appErr.Code != CodeNotFound {
		t.Errorf("Code = %q, want %q", appErr.Code, CodeNotFound)
	}
}

func TestConstructors(t *testing.T) {
	tests := []struct {
		name       string
		err        *AppError
		wantCode   string
		wantStatus int
	}{
		{
			name:       "validation",
			err:        NewValidationError("bad input", []FieldError{{Field: "email", Message: "required"}}),
			wantCode:   CodeValidation,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unauthorized",
			err:        NewUnauthorized("invalid token"),
			wantCode:   CodeUnauthorized,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "forbidden",
			err:        NewForbidden("admin only"),
			wantCode:   CodeForbidden,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "not found",
			err:        NewNotFound("project"),
			wantCode:   CodeNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "conflict",
			err:        NewConflict("email taken"),
			wantCode:   CodeConflict,
			wantStatus: http.StatusConflict,
		},
		{
			name:       "rate limited",
			err:        NewRateLimited(),
			wantCode:   CodeRateLimited,
			wantStatus: http.StatusTooManyRequests,
		},
		{
			name:       "setup required",
			err:        NewSetupRequired(),
			wantCode:   CodeSetupRequired,
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "internal",
			err:        NewInternal(fmt.Errorf("oops")),
			wantCode:   CodeInternal,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", tt.err.Code, tt.wantCode)
			}
			if tt.err.Status != tt.wantStatus {
				t.Errorf("Status = %d, want %d", tt.err.Status, tt.wantStatus)
			}
			if tt.err.Message == "" {
				t.Error("Message is empty")
			}
		})
	}
}

func TestValidationErrorDetails(t *testing.T) {
	details := []FieldError{
		{Field: "email", Message: "required"},
		{Field: "password", Message: "too short"},
	}
	e := NewValidationError("validation failed", details)
	if len(e.Details) != 2 {
		t.Fatalf("Details length = %d, want 2", len(e.Details))
	}
	if e.Details[0].Field != "email" {
		t.Errorf("Details[0].Field = %q, want %q", e.Details[0].Field, "email")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
