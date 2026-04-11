package worker

import (
	"encoding/json"
	"sync"
	"time"
)

// SSEEventType categorizes SSE events.
type SSEEventType string

const (
	SSEEventLog    SSEEventType = "log"
	SSEEventStatus SSEEventType = "status"
	SSEEventDone   SSEEventType = "done"
)

// SSEEvent is a single server-sent event.
type SSEEvent struct {
	ID        int64        `json:"id"`
	Type      SSEEventType `json:"type"`
	Data      string       `json:"data"`
	Timestamp time.Time    `json:"timestamp"`
}

// SSEHub manages per-deployment pub/sub channels for live log streaming.
type SSEHub struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan SSEEvent]struct{}
	eventID     map[string]int64
}

// NewSSEHub creates an initialized SSE hub.
func NewSSEHub() *SSEHub {
	return &SSEHub{
		subscribers: make(map[string]map[chan SSEEvent]struct{}),
		eventID:     make(map[string]int64),
	}
}

// Subscribe returns a buffered channel of SSE events for the given deployment
// and an unsubscribe function that MUST be called when done.
func (h *SSEHub) Subscribe(deploymentID string) (<-chan SSEEvent, func()) {
	ch := make(chan SSEEvent, 100)

	h.mu.Lock()
	if h.subscribers[deploymentID] == nil {
		h.subscribers[deploymentID] = make(map[chan SSEEvent]struct{})
	}
	h.subscribers[deploymentID][ch] = struct{}{}
	h.mu.Unlock()

	unsubscribe := func() {
		h.mu.Lock()
		delete(h.subscribers[deploymentID], ch)
		if len(h.subscribers[deploymentID]) == 0 {
			delete(h.subscribers, deploymentID)
		}
		close(ch)
		h.mu.Unlock()
	}

	return ch, unsubscribe
}

// Publish sends an event to all subscribers of the given deployment.
// Non-blocking: slow clients with full buffers have events dropped.
func (h *SSEHub) Publish(deploymentID string, eventType SSEEventType, data string) {
	h.mu.Lock()
	h.eventID[deploymentID]++
	id := h.eventID[deploymentID]
	h.mu.Unlock()

	event := SSEEvent{
		ID:        id,
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now().UTC(),
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.subscribers[deploymentID] {
		select {
		case ch <- event:
		default:
		}
	}
}

// PublishJSON marshals data to JSON and publishes it.
func (h *SSEHub) PublishJSON(deploymentID string, eventType SSEEventType, v interface{}) {
	data, _ := json.Marshal(v)
	h.Publish(deploymentID, eventType, string(data))
}

// Cleanup removes all state for a deployment.
func (h *SSEHub) Cleanup(deploymentID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.eventID, deploymentID)
}
