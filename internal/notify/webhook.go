package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WebhookNotifier sends notifications via HTTP webhooks.
type WebhookNotifier struct {
	url     string
	headers map[string]string
	client  *http.Client
}

// WebhookPayload is the JSON payload sent to webhooks.
type WebhookPayload struct {
	Type      string            `json:"type"`
	Project   string            `json:"project"`
	Service   string            `json:"service,omitempty"`
	Status    string            `json:"status,omitempty"`
	Message   string            `json:"message"`
	Timestamp string            `json:"timestamp"`
	Details   map[string]string `json:"details,omitempty"`
}

// NewWebhookNotifier creates a new webhook notifier.
func NewWebhookNotifier(url string, headers map[string]string) *WebhookNotifier {
	return &WebhookNotifier{
		url:     url,
		headers: headers,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the notifier name.
func (w *WebhookNotifier) Name() string {
	return "webhook"
}

// Send sends a notification via HTTP POST.
func (w *WebhookNotifier) Send(ctx context.Context, event Event) error {
	payload := WebhookPayload{
		Type:      string(event.Type),
		Project:   event.Project,
		Service:   event.Service,
		Status:    event.Status,
		Message:   FormatMessage(event),
		Timestamp: event.Timestamp.Format(time.RFC3339),
		Details:   event.Details,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range w.headers {
		req.Header.Set(k, v)
	}

	resp, err := retryableSend(ctx, w.client, req, 2)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// Close cleans up resources.
func (w *WebhookNotifier) Close() error {
	w.client.CloseIdleConnections()
	return nil
}
