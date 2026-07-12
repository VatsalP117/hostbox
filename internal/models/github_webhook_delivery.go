package models

import "time"

type GitHubWebhookDeliveryStatus string

const (
	GitHubWebhookDeliveryQueued     GitHubWebhookDeliveryStatus = "queued"
	GitHubWebhookDeliveryProcessing GitHubWebhookDeliveryStatus = "processing"
	GitHubWebhookDeliveryCompleted  GitHubWebhookDeliveryStatus = "completed"
	GitHubWebhookDeliveryFailed     GitHubWebhookDeliveryStatus = "failed"
)

// GitHubWebhookDelivery is the durable queue record for one GitHub delivery.
// DeliveryID is supplied by GitHub and is the idempotency key.
type GitHubWebhookDelivery struct {
	DeliveryID          string                      `db:"delivery_id"`
	EventType           string                      `db:"event_type"`
	Payload             []byte                      `db:"payload"`
	Status              GitHubWebhookDeliveryStatus `db:"status"`
	Attempts            int                         `db:"attempts"`
	NextAttemptAt       time.Time                   `db:"next_attempt_at"`
	LastError           *string                     `db:"last_error"`
	ProcessingStartedAt *time.Time                  `db:"processing_started_at"`
	CompletedAt         *time.Time                  `db:"completed_at"`
	CreatedAt           time.Time                   `db:"created_at"`
	UpdatedAt           time.Time                   `db:"updated_at"`
}
