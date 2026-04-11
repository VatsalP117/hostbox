package dto

import (
	"testing"
)

func TestValidateRegisterRequest(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		req     RegisterRequest
		wantErr bool
	}{
		{
			name:    "valid",
			req:     RegisterRequest{Email: "test@example.com", Password: "password123"},
			wantErr: false,
		},
		{
			name:    "missing email",
			req:     RegisterRequest{Password: "password123"},
			wantErr: true,
		},
		{
			name:    "invalid email",
			req:     RegisterRequest{Email: "notanemail", Password: "password123"},
			wantErr: true,
		},
		{
			name:    "short password",
			req:     RegisterRequest{Email: "test@example.com", Password: "short"},
			wantErr: true,
		},
		{
			name:    "missing password",
			req:     RegisterRequest{Email: "test@example.com"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateStruct(v, tt.req)
			if (len(errs) > 0) != tt.wantErr {
				t.Errorf("ValidateStruct() errors = %v, wantErr %v", errs, tt.wantErr)
			}
		})
	}
}

func TestValidateCreateProjectRequest(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		req     CreateProjectRequest
		wantErr bool
	}{
		{
			name:    "valid minimal",
			req:     CreateProjectRequest{Name: "My Project"},
			wantErr: false,
		},
		{
			name:    "missing name",
			req:     CreateProjectRequest{},
			wantErr: true,
		},
		{
			name: "valid with node version",
			req: CreateProjectRequest{
				Name:        "My Project",
				NodeVersion: strPtr("20"),
			},
			wantErr: false,
		},
		{
			name: "invalid node version",
			req: CreateProjectRequest{
				Name:        "My Project",
				NodeVersion: strPtr("16"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateStruct(v, tt.req)
			if (len(errs) > 0) != tt.wantErr {
				t.Errorf("ValidateStruct() errors = %v, wantErr %v", errs, tt.wantErr)
			}
		})
	}
}

func TestPaginationDefaults(t *testing.T) {
	p := PaginationQuery{}
	if p.PageOrDefault() != 1 {
		t.Errorf("PageOrDefault() = %d, want 1", p.PageOrDefault())
	}
	if p.PerPageOrDefault() != 20 {
		t.Errorf("PerPageOrDefault() = %d, want 20", p.PerPageOrDefault())
	}
	if p.Offset() != 0 {
		t.Errorf("Offset() = %d, want 0", p.Offset())
	}
}

func TestPaginationCustom(t *testing.T) {
	p := PaginationQuery{Page: 3, PerPage: 10}
	if p.PageOrDefault() != 3 {
		t.Errorf("PageOrDefault() = %d, want 3", p.PageOrDefault())
	}
	if p.PerPageOrDefault() != 10 {
		t.Errorf("PerPageOrDefault() = %d, want 10", p.PerPageOrDefault())
	}
	if p.Offset() != 20 {
		t.Errorf("Offset() = %d, want 20", p.Offset())
	}
}

func TestNewPaginationResponse(t *testing.T) {
	r := NewPaginationResponse(95, 2, 20)
	if r.Total != 95 {
		t.Errorf("Total = %d, want 95", r.Total)
	}
	if r.TotalPages != 5 {
		t.Errorf("TotalPages = %d, want 5", r.TotalPages)
	}
}

func TestNewPaginationResponseExact(t *testing.T) {
	r := NewPaginationResponse(40, 1, 20)
	if r.TotalPages != 2 {
		t.Errorf("TotalPages = %d, want 2", r.TotalPages)
	}
}

func TestValidateEnvVarScope(t *testing.T) {
	v := NewValidator()

	valid := CreateEnvVarRequest{Key: "API_KEY", Value: "test", Scope: strPtr("production")}
	if errs := ValidateStruct(v, valid); len(errs) > 0 {
		t.Errorf("valid scope rejected: %v", errs)
	}

	invalid := CreateEnvVarRequest{Key: "API_KEY", Value: "test", Scope: strPtr("staging")}
	if errs := ValidateStruct(v, invalid); len(errs) == 0 {
		t.Error("invalid scope 'staging' should be rejected")
	}
}

func TestValidateNotification(t *testing.T) {
	v := NewValidator()

	valid := CreateNotificationRequest{Channel: "discord", WebhookURL: "https://discord.com/api/webhooks/123"}
	if errs := ValidateStruct(v, valid); len(errs) > 0 {
		t.Errorf("valid notification rejected: %v", errs)
	}

	invalid := CreateNotificationRequest{Channel: "telegram", WebhookURL: "https://example.com"}
	if errs := ValidateStruct(v, invalid); len(errs) == 0 {
		t.Error("invalid channel 'telegram' should be rejected")
	}
}

func strPtr(s string) *string {
	return &s
}
