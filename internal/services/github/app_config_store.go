package github

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/VatsalP117/hostbox/internal/repository"
	"github.com/VatsalP117/hostbox/internal/util"
)

const (
	SettingGitHubAppID         = "github_app_id"
	SettingGitHubAppSlug       = "github_app_slug"
	SettingGitHubAppPEM        = "github_app_pem_encrypted"
	SettingGitHubWebhookSecret = "github_webhook_secret_encrypted"
	SettingGitHubManifestState = "github_manifest_state"
)

type AppConfigStore struct {
	settings      *repository.SettingsRepository
	encryptionKey string
}

func NewAppConfigStore(settings *repository.SettingsRepository, encryptionKey string) *AppConfigStore {
	return &AppConfigStore{
		settings:      settings,
		encryptionKey: encryptionKey,
	}
}

func (s *AppConfigStore) Load(ctx context.Context, envConfig AppConfig) (AppConfig, bool, error) {
	if envConfig.AppID > 0 && len(envConfig.PrivateKeyPEM) > 0 {
		return envConfig, true, nil
	}

	appIDRaw, err := s.get(ctx, SettingGitHubAppID)
	if err != nil || appIDRaw == "" {
		return AppConfig{}, false, err
	}

	appID, err := strconv.ParseInt(appIDRaw, 10, 64)
	if err != nil {
		return AppConfig{}, false, fmt.Errorf("parse stored github app id: %w", err)
	}

	appSlug, err := s.get(ctx, SettingGitHubAppSlug)
	if err != nil {
		return AppConfig{}, false, err
	}
	pemEncrypted, err := s.get(ctx, SettingGitHubAppPEM)
	if err != nil {
		return AppConfig{}, false, err
	}
	webhookSecretEncrypted, err := s.get(ctx, SettingGitHubWebhookSecret)
	if err != nil {
		return AppConfig{}, false, err
	}

	pem, err := util.Decrypt(pemEncrypted, s.encryptionKey)
	if err != nil {
		return AppConfig{}, false, fmt.Errorf("decrypt github app private key: %w", err)
	}
	webhookSecret, err := util.Decrypt(webhookSecretEncrypted, s.encryptionKey)
	if err != nil {
		return AppConfig{}, false, fmt.Errorf("decrypt github webhook secret: %w", err)
	}

	return AppConfig{
		AppID:         appID,
		AppSlug:       appSlug,
		PrivateKeyPEM: []byte(pem),
		WebhookSecret: webhookSecret,
	}, true, nil
}

func (s *AppConfigStore) Save(ctx context.Context, cfg AppConfig) error {
	pemEncrypted, err := util.Encrypt(string(cfg.PrivateKeyPEM), s.encryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt github app private key: %w", err)
	}
	webhookSecretEncrypted, err := util.Encrypt(cfg.WebhookSecret, s.encryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt github webhook secret: %w", err)
	}

	if err := s.settings.Set(ctx, SettingGitHubAppID, strconv.FormatInt(cfg.AppID, 10)); err != nil {
		return err
	}
	if err := s.settings.Set(ctx, SettingGitHubAppSlug, cfg.AppSlug); err != nil {
		return err
	}
	if err := s.settings.Set(ctx, SettingGitHubAppPEM, pemEncrypted); err != nil {
		return err
	}
	if err := s.settings.Set(ctx, SettingGitHubWebhookSecret, webhookSecretEncrypted); err != nil {
		return err
	}
	return nil
}

func (s *AppConfigStore) SetManifestState(ctx context.Context, state string) error {
	return s.settings.Set(ctx, SettingGitHubManifestState, state)
}

func (s *AppConfigStore) GetManifestState(ctx context.Context) (string, error) {
	return s.get(ctx, SettingGitHubManifestState)
}

func (s *AppConfigStore) get(ctx context.Context, key string) (string, error) {
	value, err := s.settings.Get(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return value, err
}
