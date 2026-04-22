package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ManifestConversion struct {
	ID            int64  `json:"id"`
	Slug          string `json:"slug"`
	PEM           string `json:"pem"`
	WebhookSecret string `json:"webhook_secret"`
}

func ConvertManifest(ctx context.Context, code string) (*ManifestConversion, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.github.com/app-manifests/"+code+"/conversions", nil)
	if err != nil {
		return nil, fmt.Errorf("create manifest conversion request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("convert github app manifest: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("manifest conversion returned %d: %s", resp.StatusCode, string(body))
	}

	var conversion ManifestConversion
	if err := json.Unmarshal(body, &conversion); err != nil {
		return nil, fmt.Errorf("decode manifest conversion: %w", err)
	}
	return &conversion, nil
}
