package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/VatsalP117/hostbox/internal/config"
	"github.com/VatsalP117/hostbox/internal/dto"
	apperrors "github.com/VatsalP117/hostbox/internal/errors"
	"github.com/VatsalP117/hostbox/internal/models"
	"github.com/VatsalP117/hostbox/internal/repository"
)

// JWTClaims are the custom claims embedded in access tokens.
type JWTClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
	Admin bool   `json:"admin"`
}

// AuthService handles authentication logic: JWT, sessions, registration, login, etc.
type AuthService struct {
	userRepo     *repository.UserRepository
	sessionRepo  *repository.SessionRepository
	settingsRepo *repository.SettingsRepository
	activityRepo *repository.ActivityRepository
	config       *config.Config
	logger       *slog.Logger
}

// NewAuthService creates an AuthService with the given dependencies.
func NewAuthService(
	userRepo *repository.UserRepository,
	sessionRepo *repository.SessionRepository,
	settingsRepo *repository.SettingsRepository,
	activityRepo *repository.ActivityRepository,
	cfg *config.Config,
	logger *slog.Logger,
) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		settingsRepo: settingsRepo,
		activityRepo: activityRepo,
		config:       cfg,
		logger:       logger,
	}
}

// --- JWT ---

// GenerateAccessToken creates a signed HS256 JWT for the given user.
func (s *AuthService) GenerateAccessToken(user *models.User) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.AccessTokenTTL)),
		},
		Email: user.Email,
		Admin: user.IsAdmin,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

// ValidateAccessToken parses and validates a JWT string. Returns claims or error.
func (s *AuthService) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}

// --- Refresh Token ---

// GenerateRefreshToken creates a random 32-byte token.
// Returns: raw token (base64url, for cookie) and SHA-256 hash (hex, for DB).
func (s *AuthService) GenerateRefreshToken() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	raw := base64.RawURLEncoding.EncodeToString(b)
	hash := sha256.Sum256([]byte(raw))
	return raw, hex.EncodeToString(hash[:]), nil
}

// hashToken returns the hex-encoded SHA-256 of a token string.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// --- Auth Flows ---

// Register creates a new user account with session.
func (s *AuthService) Register(ctx context.Context, req *dto.RegisterRequest, userAgent, ip string) (*models.User, string, string, error) {
	// Check if registration is enabled
	regEnabled, err := s.settingsRepo.GetBool(ctx, "registration_enabled")
	if err != nil {
		return nil, "", "", apperrors.NewInternal(err)
	}
	if !regEnabled {
		return nil, "", "", apperrors.NewForbidden("Registration is disabled")
	}

	// Check email uniqueness
	_, err = s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil {
		return nil, "", "", apperrors.NewConflict("Email already registered")
	}
	if err != sql.ErrNoRows {
		return nil, "", "", apperrors.NewInternal(err)
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", "", apperrors.NewInternal(err)
	}

	user := &models.User{
		Email:         req.Email,
		PasswordHash:  string(hash),
		DisplayName:   req.DisplayName,
		IsAdmin:       false,
		EmailVerified: true, // auto-verify when no SMTP
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, "", "", apperrors.NewInternal(err)
	}

	// Create session
	accessToken, rawRefresh, err := s.createSession(ctx, user, userAgent, ip)
	if err != nil {
		return nil, "", "", err
	}

	s.logActivity(ctx, &user.ID, "user.registered", "user", &user.ID, nil)

	return user, accessToken, rawRefresh, nil
}

// Login authenticates by email+password and creates a session.
func (s *AuthService) Login(ctx context.Context, req *dto.LoginRequest, userAgent, ip string) (*models.User, string, string, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", "", apperrors.NewUnauthorized("Invalid email or password")
		}
		return nil, "", "", apperrors.NewInternal(err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, "", "", apperrors.NewUnauthorized("Invalid email or password")
	}

	accessToken, rawRefresh, err := s.createSession(ctx, user, userAgent, ip)
	if err != nil {
		return nil, "", "", err
	}

	s.logActivity(ctx, &user.ID, "user.login", "user", &user.ID, nil)

	return user, accessToken, rawRefresh, nil
}

// CreateSession creates a new access/refresh-token session for an existing user.
func (s *AuthService) CreateSession(ctx context.Context, user *models.User, userAgent, ip string) (string, string, error) {
	return s.createSession(ctx, user, userAgent, ip)
}

// Refresh validates a refresh token and issues a new access token.
func (s *AuthService) Refresh(ctx context.Context, rawRefreshToken string) (string, error) {
	tokenHash := hashToken(rawRefreshToken)
	session, err := s.sessionRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", apperrors.NewUnauthorized("Invalid refresh token")
		}
		return "", apperrors.NewInternal(err)
	}

	if time.Now().After(session.ExpiresAt) {
		_ = s.sessionRepo.DeleteByID(ctx, session.ID)
		return "", apperrors.NewUnauthorized("Session expired")
	}

	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return "", apperrors.NewUnauthorized("User not found")
	}

	accessToken, err := s.GenerateAccessToken(user)
	if err != nil {
		return "", apperrors.NewInternal(err)
	}
	return accessToken, nil
}

// Logout invalidates a single session by its refresh token.
func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string, userID *string) error {
	tokenHash := hashToken(rawRefreshToken)
	if err := s.sessionRepo.DeleteByTokenHash(ctx, tokenHash); err != nil {
		if err == sql.ErrNoRows {
			return nil // already deleted, that's fine
		}
		return apperrors.NewInternal(err)
	}
	s.logActivity(ctx, userID, "user.logout", "user", userID, nil)
	return nil
}

// LogoutAll invalidates all sessions for a user.
func (s *AuthService) LogoutAll(ctx context.Context, userID string) (int, error) {
	count, err := s.sessionRepo.DeleteByUserID(ctx, userID)
	if err != nil {
		return 0, apperrors.NewInternal(err)
	}
	s.logActivity(ctx, &userID, "user.logout_all", "user", &userID, nil)
	return int(count), nil
}

// GetCurrentUser returns the user for the given ID.
func (s *AuthService) GetCurrentUser(ctx context.Context, userID string) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.NewNotFound("User")
		}
		return nil, apperrors.NewInternal(err)
	}
	return user, nil
}

// UpdateProfile updates the current user's display_name and/or email.
func (s *AuthService) UpdateProfile(ctx context.Context, userID string, req *dto.UpdateProfileRequest) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperrors.NewInternal(err)
	}

	if req.DisplayName != nil {
		user.DisplayName = req.DisplayName
	}
	if req.Email != nil && *req.Email != user.Email {
		// Check uniqueness of new email
		existing, err := s.userRepo.GetByEmail(ctx, *req.Email)
		if err == nil && existing.ID != user.ID {
			return nil, apperrors.NewConflict("Email already in use")
		}
		user.Email = *req.Email
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, apperrors.NewInternal(err)
	}
	return user, nil
}

// ChangePassword verifies current password and updates to new one.
func (s *AuthService) ChangePassword(ctx context.Context, userID string, req *dto.ChangePasswordRequest) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return apperrors.NewInternal(err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return apperrors.NewUnauthorized("Current password is incorrect")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperrors.NewInternal(err)
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return apperrors.NewInternal(err)
	}
	return nil
}

// ForgotPassword initiates password reset. Always returns nil to prevent email enumeration.
func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil // don't reveal whether email exists
	}

	// Generate reset token
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		s.logger.Error("failed to generate reset token", "error", err)
		return nil
	}
	rawToken := base64.RawURLEncoding.EncodeToString(b)
	tokenHash := hashToken(rawToken)
	expiresAt := time.Now().Add(1 * time.Hour)

	if err := s.userRepo.SetResetToken(ctx, user.ID, tokenHash, expiresAt); err != nil {
		s.logger.Error("failed to set reset token", "error", err)
		return nil
	}

	// TODO: Send email if SMTP configured
	s.logger.Warn("password reset requested but SMTP not configured",
		"user_id", user.ID,
		"reset_token", rawToken,
	)

	s.logActivity(ctx, &user.ID, "user.forgot_password", "user", &user.ID, nil)
	return nil
}

// ResetPassword completes password reset with a token.
func (s *AuthService) ResetPassword(ctx context.Context, req *dto.ResetPasswordRequest) error {
	tokenHash := hashToken(req.Token)
	user, err := s.userRepo.GetByResetTokenHash(ctx, tokenHash)
	if err != nil {
		return apperrors.NewValidationError("Invalid or expired reset token", nil)
	}

	if user.ResetTokenExpiresAt == nil || time.Now().After(*user.ResetTokenExpiresAt) {
		_ = s.userRepo.ClearResetToken(ctx, user.ID)
		return apperrors.NewValidationError("Reset token has expired", nil)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperrors.NewInternal(err)
	}

	if err := s.userRepo.UpdatePassword(ctx, user.ID, string(hash)); err != nil {
		return apperrors.NewInternal(err)
	}
	if err := s.userRepo.ClearResetToken(ctx, user.ID); err != nil {
		s.logger.Error("failed to clear reset token", "error", err)
	}

	// Invalidate all sessions (force re-login)
	_, _ = s.sessionRepo.DeleteByUserID(ctx, user.ID)

	s.logActivity(ctx, &user.ID, "user.reset_password", "user", &user.ID, nil)
	return nil
}

// VerifyEmail confirms a user's email address using a verification token.
func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	tokenHash := hashToken(token)
	user, err := s.userRepo.GetByEmailVerificationTokenHash(ctx, tokenHash)
	if err != nil {
		return apperrors.NewValidationError("Invalid or expired verification token", nil)
	}

	if user.EmailVerificationTokenExpiresAt == nil || time.Now().After(*user.EmailVerificationTokenExpiresAt) {
		_ = s.userRepo.ClearEmailVerificationToken(ctx, user.ID)
		return apperrors.NewValidationError("Verification token has expired", nil)
	}

	if err := s.userRepo.ClearEmailVerificationToken(ctx, user.ID); err != nil {
		return apperrors.NewInternal(err)
	}
	return nil
}

// CleanupExpiredSessions deletes all expired sessions.
func (s *AuthService) CleanupExpiredSessions(ctx context.Context) (int, error) {
	count, err := s.sessionRepo.DeleteExpired(ctx)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// --- Helpers ---

func (s *AuthService) createSession(ctx context.Context, user *models.User, userAgent, ip string) (string, string, error) {
	accessToken, err := s.GenerateAccessToken(user)
	if err != nil {
		return "", "", apperrors.NewInternal(err)
	}

	rawRefresh, tokenHash, err := s.GenerateRefreshToken()
	if err != nil {
		return "", "", apperrors.NewInternal(err)
	}

	var ua, ipAddr *string
	if userAgent != "" {
		ua = &userAgent
	}
	if ip != "" {
		ipAddr = &ip
	}

	session := &models.Session{
		UserID:           user.ID,
		RefreshTokenHash: tokenHash,
		UserAgent:        ua,
		IPAddress:        ipAddr,
		ExpiresAt:        time.Now().Add(s.config.RefreshTokenTTL),
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return "", "", apperrors.NewInternal(err)
	}

	return accessToken, rawRefresh, nil
}

func (s *AuthService) logActivity(ctx context.Context, userID *string, action, resourceType string, resourceID *string, metadata *string) {
	entry := &models.ActivityLog{
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Metadata:     metadata,
	}
	if err := s.activityRepo.Create(ctx, entry); err != nil {
		s.logger.Error("failed to log activity", "error", err, "action", action)
	}
}
