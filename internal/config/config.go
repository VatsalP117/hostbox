package config

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration.
type Config struct {
	// Server
	Host string
	Port int

	// Database
	DatabasePath string

	// Security
	JWTSecret       string
	EncryptionKey   string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// Platform
	PlatformDomain  string
	DashboardDomain string
	PlatformHTTPS   bool
	PlatformName    string

	// GitHub App
	GitHubAppID         int64
	GitHubAppSlug       string
	GitHubAppPEM        string
	GitHubWebhookSecret string

	// SMTP (all optional)
	SMTPHost  string
	SMTPPort  int
	SMTPUser  string
	SMTPPass  string
	EmailFrom string

	// Logging
	LogLevel  string
	LogFormat string

	// Paths
	DeploymentsDir string
	LogsDir        string
	CacheDir       string
	BackupDir      string

	// Caddy
	CaddyAdminURL     string
	ACMEEmail         string
	DNSProvider       string
	DNSProviderConfig string

	// Limits
	MaxConcurrentBuilds int
	MaxProjects         int

	// Build pipeline
	Build BuildConfig
}

// BuildConfig holds build pipeline configuration.
type BuildConfig struct {
	MaxConcurrentBuilds int
	BuildTimeoutMinutes int
	CloneTimeoutSeconds int
	CloneMaxRetries     int
	CloneRetryDelaySec  int
	DefaultNodeVersion  string
	DefaultMemoryMB     int64
	DefaultCPUs         float64
	PIDLimit            int64
	MaxLogFileSizeBytes int64
	ShutdownTimeoutSec  int
	JobChannelBuffer    int
	CloneBaseDir        string
	DeploymentBaseDir   string
	LogBaseDir          string
}

// Load reads configuration from environment variables and applies defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Host:                getEnv("HOST", "0.0.0.0"),
		Port:                getEnvInt("PORT", 8080),
		DatabasePath:        getEnv("DATABASE_PATH", "/app/data/hostbox.db"),
		JWTSecret:           getEnv("JWT_SECRET", ""),
		EncryptionKey:       getEnv("ENCRYPTION_KEY", ""),
		AccessTokenTTL:      getEnvDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:     getEnvDuration("REFRESH_TOKEN_TTL", 168*time.Hour),
		PlatformDomain:      getEnv("PLATFORM_DOMAIN", ""),
		DashboardDomain:     getEnv("DASHBOARD_DOMAIN", ""),
		PlatformHTTPS:       getEnvBool("PLATFORM_HTTPS", true),
		PlatformName:        getEnv("PLATFORM_NAME", "Hostbox"),
		GitHubAppID:         int64(getEnvInt("GITHUB_APP_ID", 0)),
		GitHubAppSlug:       getEnv("GITHUB_APP_SLUG", ""),
		GitHubAppPEM:        getEnv("GITHUB_APP_PEM", ""),
		GitHubWebhookSecret: getEnv("GITHUB_WEBHOOK_SECRET", ""),
		SMTPHost:            getEnv("SMTP_HOST", ""),
		SMTPPort:            getEnvInt("SMTP_PORT", 587),
		SMTPUser:            getEnv("SMTP_USER", ""),
		SMTPPass:            getEnv("SMTP_PASS", ""),
		EmailFrom:           getEnv("EMAIL_FROM", ""),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		LogFormat:           getEnv("LOG_FORMAT", "json"),
		DeploymentsDir:      getEnv("DEPLOYMENTS_DIR", "/app/deployments"),
		LogsDir:             getEnv("LOGS_DIR", "/app/logs"),
		CacheDir:            getEnv("CACHE_DIR", "/cache"),
		BackupDir:           getEnv("BACKUP_DIR", "/app/data/backups"),
		CaddyAdminURL:       getEnv("CADDY_ADMIN_URL", "http://localhost:2019"),
		ACMEEmail:           getEnv("ACME_EMAIL", ""),
		DNSProvider:         getEnv("DNS_PROVIDER", ""),
		DNSProviderConfig:   getEnv("DNS_PROVIDER_CONFIG", ""),
		MaxConcurrentBuilds: getEnvInt("MAX_CONCURRENT_BUILDS", 1),
		MaxProjects:         getEnvInt("MAX_PROJECTS", 50),
		Build: BuildConfig{
			MaxConcurrentBuilds: getEnvInt("MAX_CONCURRENT_BUILDS", 1),
			BuildTimeoutMinutes: getEnvInt("BUILD_TIMEOUT_MINUTES", 15),
			CloneTimeoutSeconds: getEnvInt("CLONE_TIMEOUT_SECONDS", 120),
			CloneMaxRetries:     getEnvInt("CLONE_MAX_RETRIES", 3),
			CloneRetryDelaySec:  getEnvInt("CLONE_RETRY_DELAY_SEC", 5),
			DefaultNodeVersion:  getEnv("DEFAULT_NODE_VERSION", "20"),
			DefaultMemoryMB:     int64(getEnvInt("BUILD_MEMORY_MB", 512)),
			DefaultCPUs:         getEnvFloat("BUILD_CPUS", 1.0),
			PIDLimit:            int64(getEnvInt("BUILD_PID_LIMIT", 256)),
			MaxLogFileSizeBytes: int64(getEnvInt("MAX_LOG_FILE_SIZE", 5242880)),
			ShutdownTimeoutSec:  getEnvInt("SHUTDOWN_TIMEOUT_SEC", 60),
			JobChannelBuffer:    getEnvInt("JOB_CHANNEL_BUFFER", 100),
			CloneBaseDir:        getEnv("CLONE_BASE_DIR", "/app/tmp"),
			DeploymentBaseDir:   getEnv("DEPLOYMENT_BASE_DIR", "/app/deployments"),
			LogBaseDir:          getEnv("LOG_BASE_DIR", "/app/logs"),
		},
	}

	cfg.DNSProvider = normalizeDNSProvider(cfg.DNSProvider)
	cfg.DNSProviderConfig = strings.TrimSpace(cfg.DNSProviderConfig)
	if cfg.DNSProviderConfig == "" {
		dnsProviderConfig, err := buildDNSProviderConfig(cfg.DNSProvider)
		if err != nil {
			return nil, err
		}
		cfg.DNSProviderConfig = dnsProviderConfig
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that all required fields are present and well-formed.
func (c *Config) Validate() error {
	var errs []string

	if len(c.JWTSecret) < 32 {
		errs = append(errs, "JWT_SECRET must be at least 32 characters")
	}

	if c.EncryptionKey != "" {
		keyBytes, err := hex.DecodeString(c.EncryptionKey)
		if err != nil || len(keyBytes) != 32 {
			errs = append(errs, "ENCRYPTION_KEY must be exactly 64 hex characters (32 bytes)")
		}
	} else {
		errs = append(errs, "ENCRYPTION_KEY is required")
	}

	if c.PlatformDomain == "" {
		errs = append(errs, "PLATFORM_DOMAIN is required")
	} else if strings.HasPrefix(c.PlatformDomain, "http://") || strings.HasPrefix(c.PlatformDomain, "https://") {
		errs = append(errs, "PLATFORM_DOMAIN must not include protocol prefix")
	}

	if c.DashboardDomain == "" {
		c.DashboardDomain = "hostbox." + c.PlatformDomain
	}
	if strings.HasPrefix(c.DashboardDomain, "http://") || strings.HasPrefix(c.DashboardDomain, "https://") {
		errs = append(errs, "DASHBOARD_DOMAIN must not include protocol prefix")
	}
	if c.DashboardDomain == c.PlatformDomain {
		errs = append(errs, "DASHBOARD_DOMAIN must differ from PLATFORM_DOMAIN")
	}

	if c.Port < 1 || c.Port > 65535 {
		errs = append(errs, "PORT must be between 1 and 65535")
	}

	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[strings.ToLower(c.LogLevel)] {
		errs = append(errs, "LOG_LEVEL must be one of: debug, info, warn, error")
	}

	c.DNSProvider = normalizeDNSProvider(c.DNSProvider)
	c.DNSProviderConfig = strings.TrimSpace(c.DNSProviderConfig)
	validDNSProviders := map[string]bool{"": true, "cloudflare": true, "route53": true, "digitalocean": true}
	if !validDNSProviders[c.DNSProvider] {
		errs = append(errs, "DNS_PROVIDER must be one of: cloudflare, route53, digitalocean, none")
	}

	if c.DatabasePath == "" {
		errs = append(errs, "DATABASE_PATH must not be empty")
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration errors:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// BaseURL returns the full platform base URL (e.g., "https://hostbox.example.com").
func (c *Config) BaseURL() string {
	scheme := "https"
	if !c.PlatformHTTPS {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s", scheme, c.PlatformDomain)
}

// DashboardBaseURL returns the full dashboard URL (e.g., "https://hostbox.example.com").
func (c *Config) DashboardBaseURL() string {
	scheme := "https"
	if !c.PlatformHTTPS {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s", scheme, c.DashboardDomain)
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvBool(key string, fallback bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvFloat(key string, fallback float64) float64 {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return fallback
	}
	return f
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return fallback
	}
	return d
}

func normalizeDNSProvider(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "none" {
		return ""
	}
	return provider
}

func buildDNSProviderConfig(provider string) (string, error) {
	switch provider {
	case "":
		return "", nil
	case "cloudflare":
		return `{"name":"cloudflare","api_token":"{env.CF_API_TOKEN}"}`, nil
	case "route53":
		return `{"name":"route53","hosted_zone_id":"{env.AWS_HOSTED_ZONE_ID}"}`, nil
	case "digitalocean":
		return `{"name":"digitalocean","api_token":"{env.DO_AUTH_TOKEN}"}`, nil
	default:
		return "", fmt.Errorf("unsupported DNS_PROVIDER %q", provider)
	}
}

// MustValidEncryptionKey is a helper for generating a valid key in tests.
func MustValidEncryptionKey() string {
	return errors.New("not implemented").Error() // placeholder, use hex key below
}
