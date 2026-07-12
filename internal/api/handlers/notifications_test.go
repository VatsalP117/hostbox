package handlers_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/VatsalP117/hostbox/internal/api/handlers"
	appmiddleware "github.com/VatsalP117/hostbox/internal/api/middleware"
	apperrors "github.com/VatsalP117/hostbox/internal/errors"
	"github.com/VatsalP117/hostbox/internal/models"
	notificationsvc "github.com/VatsalP117/hostbox/internal/services/notification"
)

type notificationHandlerTestEnv struct {
	*testEnv
	handler *handlers.NotificationHandler
	user    *models.User
	project *models.Project
}

func setupNotificationHandlerTest(t *testing.T) *notificationHandlerTestEnv {
	t.Helper()
	env := setupTestEnv(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	user := &models.User{
		ID:           "notification-user",
		Email:        "notification@example.com",
		PasswordHash: "hash",
	}
	if err := env.repos.User.Create(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	project := &models.Project{
		ID:      "notification-project",
		OwnerID: user.ID,
		Name:    "Notification App",
		Slug:    "notification-app",
	}
	if err := env.repos.Project.Create(context.Background(), project); err != nil {
		t.Fatalf("create project: %v", err)
	}
	service := notificationsvc.NewService(env.repos.Notification, logger)
	return &notificationHandlerTestEnv{
		testEnv: env,
		handler: handlers.NewNotificationHandler(
			env.repos.Notification,
			env.repos.Project,
			env.repos.Activity,
			service,
			logger,
		),
		user:    user,
		project: project,
	}
}

func notificationContext(e *echo.Echo, method, target, body string, user *models.User, paramNames, paramValues []string) (echo.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(string(appmiddleware.UserContextKey), user)
	c.SetParamNames(paramNames...)
	c.SetParamValues(paramValues...)
	return c, rec
}

func requireBadWebhookURL(t *testing.T, err error) {
	t.Helper()
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("error type = %T, want *errors.AppError: %v", err, err)
	}
	if appErr.Status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", appErr.Status)
	}
	if !strings.HasPrefix(appErr.Message, "Invalid webhook URL: ") {
		t.Fatalf("message = %q, want actionable webhook validation message", appErr.Message)
	}
}

func TestNotificationCreate_ValidatesWebhookURLBeforePersistence(t *testing.T) {
	tests := []struct {
		name       string
		webhookURL string
	}{
		{name: "HTTP", webhookURL: "http://198.51.100.10/hook"},
		{name: "IPv4 loopback", webhookURL: "https://127.0.0.1/hook"},
		{name: "private IPv4", webhookURL: "https://10.20.30.40/hook"},
		{name: "IPv6 loopback", webhookURL: "https://[::1]/hook"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := setupNotificationHandlerTest(t)
			body := `{"channel":"webhook","webhook_url":"` + tt.webhookURL + `"}`
			c, _ := notificationContext(env.echo, http.MethodPost, "/notifications", body, env.user, []string{"projectId"}, []string{env.project.ID})

			requireBadWebhookURL(t, env.handler.Create(c))
			count, err := env.repos.Notification.CountByProject(context.Background(), env.project.ID)
			if err != nil {
				t.Fatalf("count notifications: %v", err)
			}
			if count != 0 {
				t.Fatalf("persisted notifications = %d, want 0", count)
			}
		})
	}
}

func TestNotificationCreate_AcceptsValidHTTPSURL(t *testing.T) {
	env := setupNotificationHandlerTest(t)
	const webhookURL = "https://198.51.100.10/hook"
	body := `{"channel":"webhook","webhook_url":"` + webhookURL + `"}`
	c, rec := notificationContext(env.echo, http.MethodPost, "/notifications", body, env.user, []string{"projectId"}, []string{env.project.ID})

	if err := env.handler.Create(c); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body.String())
	}
	configs, err := env.repos.Notification.ListByProject(context.Background(), env.project.ID)
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(configs) != 1 || configs[0].WebhookURL != webhookURL {
		t.Fatalf("persisted configs = %#v, want one valid HTTPS URL", configs)
	}
}

func TestNotificationUpdate_RejectsUnsafeURLWithoutMutation(t *testing.T) {
	env := setupNotificationHandlerTest(t)
	config := &models.NotificationConfig{
		ID:         "notification-config",
		ProjectID:  &env.project.ID,
		Channel:    "webhook",
		WebhookURL: "https://198.51.100.10/original",
		Events:     "all",
		Enabled:    true,
	}
	if err := env.repos.Notification.Create(context.Background(), config); err != nil {
		t.Fatalf("create notification: %v", err)
	}
	c, _ := notificationContext(env.echo, http.MethodPatch, "/notifications/"+config.ID, `{"webhook_url":"https://192.168.1.20/hook"}`, env.user, []string{"id"}, []string{config.ID})

	requireBadWebhookURL(t, env.handler.Update(c))
	stored, err := env.repos.Notification.GetByID(context.Background(), config.ID)
	if err != nil {
		t.Fatalf("get notification: %v", err)
	}
	if stored.WebhookURL != config.WebhookURL {
		t.Fatalf("stored URL = %q, want unchanged %q", stored.WebhookURL, config.WebhookURL)
	}
}

func TestNotificationUpdate_RejectsUnsafeStoredURLBeforePersistence(t *testing.T) {
	env := setupNotificationHandlerTest(t)
	config := &models.NotificationConfig{
		ID:         "legacy-unsafe-notification-config",
		ProjectID:  &env.project.ID,
		Channel:    "webhook",
		WebhookURL: "https://127.0.0.1/hook",
		Events:     "all",
		Enabled:    true,
	}
	if err := env.repos.Notification.Create(context.Background(), config); err != nil {
		t.Fatalf("create notification: %v", err)
	}
	c, _ := notificationContext(env.echo, http.MethodPatch, "/notifications/"+config.ID, `{"enabled":false}`, env.user, []string{"id"}, []string{config.ID})

	requireBadWebhookURL(t, env.handler.Update(c))
	stored, err := env.repos.Notification.GetByID(context.Background(), config.ID)
	if err != nil {
		t.Fatalf("get notification: %v", err)
	}
	if !stored.Enabled {
		t.Fatal("unsafe notification update was persisted")
	}
}

func TestNotificationTest_RejectsUnsafeStoredURLBeforeSend(t *testing.T) {
	env := setupNotificationHandlerTest(t)
	config := &models.NotificationConfig{
		ID:         "unsafe-notification-config",
		ProjectID:  &env.project.ID,
		Channel:    "webhook",
		WebhookURL: "https://127.0.0.1/hook",
		Events:     "all",
		Enabled:    true,
	}
	if err := env.repos.Notification.Create(context.Background(), config); err != nil {
		t.Fatalf("create notification: %v", err)
	}
	c, _ := notificationContext(env.echo, http.MethodPost, "/notifications/"+config.ID+"/test", "", env.user, []string{"id"}, []string{config.ID})

	requireBadWebhookURL(t, env.handler.Test(c))
}
