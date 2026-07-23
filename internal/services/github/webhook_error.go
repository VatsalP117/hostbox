package github

import "fmt"

// PermanentWebhookError marks a valid webhook event that cannot succeed on a
// retry, such as an unsupported fork pull request. The durable processor stores
// the reason and transitions the delivery directly to failed.
type PermanentWebhookError struct {
	message string
}

func NewPermanentWebhookError(format string, args ...any) error {
	return &PermanentWebhookError{message: fmt.Sprintf(format, args...)}
}

func (e *PermanentWebhookError) Error() string {
	return e.message
}
