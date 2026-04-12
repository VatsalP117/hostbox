package github

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func generateTestKey(t *testing.T) (*rsa.PrivateKey, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	return key, pemBytes
}

func TestNewTokenProvider(t *testing.T) {
	_, pemBytes := generateTestKey(t)

	tp, err := NewTokenProvider(AppConfig{
		AppID:         12345,
		PrivateKeyPEM: pemBytes,
	}, slog.Default())
	if err != nil {
		t.Fatalf("NewTokenProvider failed: %v", err)
	}
	if tp.appID != 12345 {
		t.Errorf("appID = %d, want 12345", tp.appID)
	}
}

func TestNewTokenProvider_InvalidPEM(t *testing.T) {
	_, err := NewTokenProvider(AppConfig{
		AppID:         12345,
		PrivateKeyPEM: []byte("not-a-pem"),
	}, slog.Default())
	if err == nil {
		t.Fatal("expected error for invalid PEM")
	}
}

func TestGenerateAppJWT(t *testing.T) {
	key, pemBytes := generateTestKey(t)

	tp, err := NewTokenProvider(AppConfig{
		AppID:         12345,
		PrivateKeyPEM: pemBytes,
	}, slog.Default())
	if err != nil {
		t.Fatalf("NewTokenProvider failed: %v", err)
	}

	tokenStr, err := tp.GenerateAppJWT()
	if err != nil {
		t.Fatalf("GenerateAppJWT failed: %v", err)
	}

	// Parse and verify
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return &key.PublicKey, nil
	})
	if err != nil {
		t.Fatalf("parse JWT failed: %v", err)
	}
	if !token.Valid {
		t.Error("JWT is not valid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("cannot extract claims")
	}
	if claims["iss"] != "12345" {
		t.Errorf("issuer = %v, want 12345", claims["iss"])
	}
}

func TestGetInstallationToken(t *testing.T) {
	_, pemBytes := generateTestKey(t)

	expiresAt := time.Now().Add(1 * time.Hour)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/app/installations/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Errorf("missing Bearer auth: %s", auth)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(installationTokenResponse{
			Token:     "ghs_test_token_123",
			ExpiresAt: expiresAt,
		})
	}))
	defer server.Close()

	tp, err := NewTokenProviderWithBaseURL(AppConfig{
		AppID:         12345,
		PrivateKeyPEM: pemBytes,
	}, slog.Default(), server.URL)
	if err != nil {
		t.Fatalf("NewTokenProvider failed: %v", err)
	}

	// First call - fetches from server
	token, err := tp.GetInstallationToken(99)
	if err != nil {
		t.Fatalf("GetInstallationToken failed: %v", err)
	}
	if token != "ghs_test_token_123" {
		t.Errorf("token = %q, want ghs_test_token_123", token)
	}

	// Second call - should be cached
	token2, err := tp.GetInstallationToken(99)
	if err != nil {
		t.Fatalf("cached GetInstallationToken failed: %v", err)
	}
	if token2 != token {
		t.Error("expected cached token")
	}
}

func TestInvalidateToken(t *testing.T) {
	_, pemBytes := generateTestKey(t)

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(installationTokenResponse{
			Token:     fmt.Sprintf("token_%d", callCount),
			ExpiresAt: time.Now().Add(1 * time.Hour),
		})
	}))
	defer server.Close()

	tp, _ := NewTokenProviderWithBaseURL(AppConfig{
		AppID:         12345,
		PrivateKeyPEM: pemBytes,
	}, slog.Default(), server.URL)

	token1, _ := tp.GetInstallationToken(99)
	tp.InvalidateToken(99)
	token2, _ := tp.GetInstallationToken(99)

	if token1 == token2 {
		t.Error("expected different token after invalidation")
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
}
