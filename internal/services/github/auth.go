package github

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AppConfig holds the GitHub App configuration.
type AppConfig struct {
	AppID         int64
	AppSlug       string
	PrivateKeyPEM []byte
	WebhookSecret string
}

// TokenProvider manages GitHub App JWT and installation token lifecycle.
type TokenProvider struct {
	appID      int64
	privateKey *rsa.PrivateKey
	httpClient *http.Client
	logger     *slog.Logger
	baseURL    string

	mu     sync.RWMutex
	tokens map[int64]*CachedToken
}

// CachedToken holds a cached installation access token.
type CachedToken struct {
	Token     string
	ExpiresAt time.Time
}

func NewTokenProvider(cfg AppConfig, logger *slog.Logger) (*TokenProvider, error) {
	return NewTokenProviderWithBaseURL(cfg, logger, "https://api.github.com")
}

func NewTokenProviderWithBaseURL(cfg AppConfig, logger *slog.Logger, baseURL string) (*TokenProvider, error) {
	block, _ := pem.Decode(cfg.PrivateKeyPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from GITHUB_APP_PEM")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		pkcs8Key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("parse private key (PKCS1: %v, PKCS8: %v)", err, err2)
		}
		var ok bool
		key, ok = pkcs8Key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}
	}

	return &TokenProvider{
		appID:      cfg.AppID,
		privateKey: key,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger,
		baseURL:    baseURL,
		tokens:     make(map[int64]*CachedToken),
	}, nil
}

// GenerateAppJWT creates a JWT signed with the App's private key (valid 10 min).
func (tp *TokenProvider) GenerateAppJWT() (string, error) {
	now := time.Now()

	claims := jwt.RegisteredClaims{
		Issuer:    fmt.Sprintf("%d", tp.appID),
		IssuedAt:  jwt.NewNumericDate(now.Add(-60 * time.Second)),
		ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(tp.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}
	return signed, nil
}

type installationTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// GetInstallationToken returns a valid installation access token (cached, refreshed 5 min before expiry).
func (tp *TokenProvider) GetInstallationToken(installationID int64) (string, error) {
	tp.mu.RLock()
	cached, exists := tp.tokens[installationID]
	tp.mu.RUnlock()

	if exists && time.Now().Add(5*time.Minute).Before(cached.ExpiresAt) {
		return cached.Token, nil
	}

	appJWT, err := tp.GenerateAppJWT()
	if err != nil {
		return "", fmt.Errorf("generate app JWT: %w", err)
	}

	url := fmt.Sprintf("%s/app/installations/%d/access_tokens", tp.baseURL, installationID)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+appJWT)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := tp.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request installation token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github returned %d creating installation token: %s", resp.StatusCode, string(body))
	}

	var tokenResp installationTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}

	tp.mu.Lock()
	tp.tokens[installationID] = &CachedToken{
		Token:     tokenResp.Token,
		ExpiresAt: tokenResp.ExpiresAt,
	}
	tp.mu.Unlock()

	tp.logger.Debug("refreshed installation token",
		"installation_id", installationID,
		"expires_at", tokenResp.ExpiresAt,
	)

	return tokenResp.Token, nil
}

// InvalidateToken removes a cached token.
func (tp *TokenProvider) InvalidateToken(installationID int64) {
	tp.mu.Lock()
	delete(tp.tokens, installationID)
	tp.mu.Unlock()
}
