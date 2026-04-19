package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// validTestEnv sets required env vars for testing.
func validTestEnv(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", "this-is-a-very-long-secret-that-is-at-least-32-chars")
	t.Setenv("ENCRYPTION_KEY", "6368616e676520746869732070617373776f726420746f206120736563726574")
	t.Setenv("PLATFORM_DOMAIN", "example.com")
}

func TestLoadDefaults(t *testing.T) {
	validTestEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Host != "0.0.0.0" {
		t.Errorf("Host = %q, want %q", cfg.Host, "0.0.0.0")
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8080)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.LogFormat != "json" {
		t.Errorf("LogFormat = %q, want %q", cfg.LogFormat, "json")
	}
	if cfg.PlatformName != "Hostbox" {
		t.Errorf("PlatformName = %q, want %q", cfg.PlatformName, "Hostbox")
	}
	if cfg.PlatformHTTPS != true {
		t.Error("PlatformHTTPS should default to true")
	}
	if cfg.DashboardDomain != "hostbox.example.com" {
		t.Errorf("DashboardDomain = %q, want %q", cfg.DashboardDomain, "hostbox.example.com")
	}
	if cfg.MaxConcurrentBuilds != 1 {
		t.Errorf("MaxConcurrentBuilds = %d, want 1", cfg.MaxConcurrentBuilds)
	}
	if cfg.CaddyAPIUpstream != "localhost:8080" {
		t.Errorf("CaddyAPIUpstream = %q, want localhost:8080", cfg.CaddyAPIUpstream)
	}
	if cfg.AccessTokenTTL != 15*time.Minute {
		t.Errorf("AccessTokenTTL = %v, want 15m", cfg.AccessTokenTTL)
	}
	if cfg.RefreshTokenTTL != 168*time.Hour {
		t.Errorf("RefreshTokenTTL = %v, want 168h", cfg.RefreshTokenTTL)
	}
	if !filepath.IsAbs(cfg.DatabasePath) {
		t.Errorf("DatabasePath should be absolute, got %q", cfg.DatabasePath)
	}
	if cfg.Build.DeploymentBaseDir != cfg.DeploymentsDir {
		t.Errorf("Build.DeploymentBaseDir = %q, want %q", cfg.Build.DeploymentBaseDir, cfg.DeploymentsDir)
	}
	if cfg.Build.LogBaseDir != cfg.LogsDir {
		t.Errorf("Build.LogBaseDir = %q, want %q", cfg.Build.LogBaseDir, cfg.LogsDir)
	}
	expectedCloneDir := filepath.Join(cfg.CacheDir, "clones")
	if cfg.Build.CloneBaseDir != expectedCloneDir {
		t.Errorf("Build.CloneBaseDir = %q, want %q", cfg.Build.CloneBaseDir, expectedCloneDir)
	}
}

func TestLoadMissingJWTSecret(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", "6368616e676520746869732070617373776f726420746f206120736563726574")
	t.Setenv("PLATFORM_DOMAIN", "hostbox.example.com")
	// JWT_SECRET not set

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should fail with missing JWT_SECRET")
	}
}

func TestLoadMissingEncryptionKey(t *testing.T) {
	t.Setenv("JWT_SECRET", "this-is-a-very-long-secret-that-is-at-least-32-chars")
	t.Setenv("PLATFORM_DOMAIN", "hostbox.example.com")
	// ENCRYPTION_KEY not set

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should fail with missing ENCRYPTION_KEY")
	}
}

func TestLoadMissingPlatformDomain(t *testing.T) {
	t.Setenv("JWT_SECRET", "this-is-a-very-long-secret-that-is-at-least-32-chars")
	t.Setenv("ENCRYPTION_KEY", "6368616e676520746869732070617373776f726420746f206120736563726574")
	// PLATFORM_DOMAIN not set

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should fail with missing PLATFORM_DOMAIN")
	}
}

func TestLoadShortJWTSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "tooshort")
	t.Setenv("ENCRYPTION_KEY", "6368616e676520746869732070617373776f726420746f206120736563726574")
	t.Setenv("PLATFORM_DOMAIN", "hostbox.example.com")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should fail with short JWT_SECRET")
	}
}

func TestLoadBadEncryptionKey(t *testing.T) {
	t.Setenv("JWT_SECRET", "this-is-a-very-long-secret-that-is-at-least-32-chars")
	t.Setenv("ENCRYPTION_KEY", "not-valid-hex")
	t.Setenv("PLATFORM_DOMAIN", "hostbox.example.com")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should fail with bad ENCRYPTION_KEY")
	}
}

func TestLoadDomainWithProtocol(t *testing.T) {
	t.Setenv("JWT_SECRET", "this-is-a-very-long-secret-that-is-at-least-32-chars")
	t.Setenv("ENCRYPTION_KEY", "6368616e676520746869732070617373776f726420746f206120736563726574")
	t.Setenv("PLATFORM_DOMAIN", "https://hostbox.example.com")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should fail when PLATFORM_DOMAIN includes protocol")
	}
}

func TestLoadDashboardDomainOverride(t *testing.T) {
	validTestEnv(t)
	t.Setenv("DASHBOARD_DOMAIN", "admin.example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.DashboardDomain != "admin.example.com" {
		t.Errorf("DashboardDomain = %q, want %q", cfg.DashboardDomain, "admin.example.com")
	}
}

func TestLoadDashboardDomainWithProtocol(t *testing.T) {
	validTestEnv(t)
	t.Setenv("DASHBOARD_DOMAIN", "https://hostbox.example.com")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should fail when DASHBOARD_DOMAIN includes protocol")
	}
}

func TestLoadDashboardDomainEqualsPlatformDomain(t *testing.T) {
	validTestEnv(t)
	t.Setenv("DASHBOARD_DOMAIN", "example.com")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should fail when DASHBOARD_DOMAIN equals PLATFORM_DOMAIN")
	}
}

func TestLoadCustomPort(t *testing.T) {
	validTestEnv(t)
	t.Setenv("PORT", "3000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Port != 3000 {
		t.Errorf("Port = %d, want 3000", cfg.Port)
	}
	if cfg.CaddyAPIUpstream != "localhost:3000" {
		t.Errorf("CaddyAPIUpstream = %q, want localhost:3000", cfg.CaddyAPIUpstream)
	}
}

func TestLoadBoolConversion(t *testing.T) {
	validTestEnv(t)
	t.Setenv("PLATFORM_HTTPS", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.PlatformHTTPS != false {
		t.Error("PlatformHTTPS should be false")
	}
}

func TestLoadCaddyAPIUpstreamOverride(t *testing.T) {
	validTestEnv(t)
	t.Setenv("CADDY_API_UPSTREAM", "hostbox:8080")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.CaddyAPIUpstream != "hostbox:8080" {
		t.Errorf("CaddyAPIUpstream = %q, want hostbox:8080", cfg.CaddyAPIUpstream)
	}
}

func TestLoadDurationConversion(t *testing.T) {
	validTestEnv(t)
	t.Setenv("ACCESS_TOKEN_TTL", "30m")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.AccessTokenTTL != 30*time.Minute {
		t.Errorf("AccessTokenTTL = %v, want 30m", cfg.AccessTokenTTL)
	}
}

func TestLoadNormalizesRelativePaths(t *testing.T) {
	validTestEnv(t)

	t.Setenv("DATABASE_PATH", "./data/hostbox.db")
	t.Setenv("DEPLOYMENTS_DIR", "./deployments")
	t.Setenv("LOGS_DIR", "./logs")
	t.Setenv("CACHE_DIR", "./tmp")
	t.Setenv("BACKUP_DIR", "./data/backups")
	t.Setenv("CLONE_BASE_DIR", "./tmp")
	t.Setenv("DEPLOYMENT_BASE_DIR", "./deployments")
	t.Setenv("LOG_BASE_DIR", "./logs")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	for name, got := range map[string]string{
		"DATABASE_PATH":       cfg.DatabasePath,
		"DEPLOYMENTS_DIR":     cfg.DeploymentsDir,
		"LOGS_DIR":            cfg.LogsDir,
		"CACHE_DIR":           cfg.CacheDir,
		"BACKUP_DIR":          cfg.BackupDir,
		"CLONE_BASE_DIR":      cfg.Build.CloneBaseDir,
		"DEPLOYMENT_BASE_DIR": cfg.Build.DeploymentBaseDir,
		"LOG_BASE_DIR":        cfg.Build.LogBaseDir,
	} {
		if !filepath.IsAbs(got) {
			t.Fatalf("%s should be absolute, got %q", name, got)
		}
	}
}

func TestLoadDNSProviderNone(t *testing.T) {
	validTestEnv(t)
	t.Setenv("DNS_PROVIDER", "none")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.DNSProvider != "" {
		t.Fatalf("DNSProvider = %q, want empty", cfg.DNSProvider)
	}
	if cfg.DNSProviderConfig != "" {
		t.Fatalf("DNSProviderConfig = %q, want empty", cfg.DNSProviderConfig)
	}
}

func TestLoadDNSProviderConfigDerived(t *testing.T) {
	validTestEnv(t)
	t.Setenv("DNS_PROVIDER", "cloudflare")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.DNSProvider != "cloudflare" {
		t.Fatalf("DNSProvider = %q, want cloudflare", cfg.DNSProvider)
	}
	expected := `{"name":"cloudflare","api_token":"{env.CF_API_TOKEN}"}`
	if cfg.DNSProviderConfig != expected {
		t.Fatalf("DNSProviderConfig = %q, want %q", cfg.DNSProviderConfig, expected)
	}
}

func TestLoadDNSProviderConfigDerivedWhenWhitespace(t *testing.T) {
	validTestEnv(t)
	t.Setenv("DNS_PROVIDER", "cloudflare")
	t.Setenv("DNS_PROVIDER_CONFIG", "   ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	expected := `{"name":"cloudflare","api_token":"{env.CF_API_TOKEN}"}`
	if cfg.DNSProviderConfig != expected {
		t.Fatalf("DNSProviderConfig = %q, want %q", cfg.DNSProviderConfig, expected)
	}
}

func TestLoadUnknownDNSProvider(t *testing.T) {
	validTestEnv(t)
	t.Setenv("DNS_PROVIDER", "unsupported")

	if _, err := Load(); err == nil {
		t.Fatal("Load() should fail for unsupported DNS provider")
	}
}

func TestValidateNormalizesDNSProvider(t *testing.T) {
	cfg := &Config{
		JWTSecret:        "this-is-a-very-long-secret-that-is-at-least-32-chars",
		EncryptionKey:    "6368616e676520746869732070617373776f726420746f206120736563726574",
		DatabasePath:     "/tmp/hostbox-test.db",
		PlatformDomain:   "example.com",
		DashboardDomain:  "hostbox.example.com",
		Port:             8080,
		LogLevel:         "info",
		DNSProvider:      "none",
		CaddyAPIUpstream: "localhost:8080",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
	if cfg.DNSProvider != "" {
		t.Fatalf("DNSProvider = %q, want empty after normalization", cfg.DNSProvider)
	}
}

func TestBaseURL(t *testing.T) {
	cfg := &Config{PlatformDomain: "example.com", PlatformHTTPS: true}
	if cfg.BaseURL() != "https://example.com" {
		t.Errorf("BaseURL() = %q, want %q", cfg.BaseURL(), "https://example.com")
	}

	cfg.PlatformHTTPS = false
	if cfg.BaseURL() != "http://example.com" {
		t.Errorf("BaseURL() = %q, want %q", cfg.BaseURL(), "http://example.com")
	}
}

func TestDashboardBaseURL(t *testing.T) {
	cfg := &Config{DashboardDomain: "hostbox.example.com", PlatformHTTPS: true}
	if cfg.DashboardBaseURL() != "https://hostbox.example.com" {
		t.Errorf("DashboardBaseURL() = %q, want %q", cfg.DashboardBaseURL(), "https://hostbox.example.com")
	}

	cfg.PlatformHTTPS = false
	if cfg.DashboardBaseURL() != "http://hostbox.example.com" {
		t.Errorf("DashboardBaseURL() = %q, want %q", cfg.DashboardBaseURL(), "http://hostbox.example.com")
	}
}

func TestGetEnvHelpers(t *testing.T) {
	// Test getEnvInt with invalid value
	t.Setenv("TEST_INT", "notanumber")
	if got := getEnvInt("TEST_INT", 42); got != 42 {
		t.Errorf("getEnvInt with invalid = %d, want 42", got)
	}

	// Test getEnvBool with invalid value
	t.Setenv("TEST_BOOL", "notabool")
	if got := getEnvBool("TEST_BOOL", true); got != true {
		t.Errorf("getEnvBool with invalid = %v, want true", got)
	}

	// Test getEnvDuration with invalid value
	t.Setenv("TEST_DUR", "notaduration")
	if got := getEnvDuration("TEST_DUR", time.Hour); got != time.Hour {
		t.Errorf("getEnvDuration with invalid = %v, want 1h", got)
	}

	// Test unset env returns fallback
	os.Unsetenv("UNSET_VAR_TEST")
	if got := getEnv("UNSET_VAR_TEST", "fallback"); got != "fallback" {
		t.Errorf("getEnv unset = %q, want %q", got, "fallback")
	}
}
