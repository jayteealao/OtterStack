package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWebhookNotifier(t *testing.T) {
	t.Run("creates notifier with correct config", func(t *testing.T) {
		url := "https://example.com/webhook"
		headers := map[string]string{
			"Authorization": "Bearer token123",
			"X-Custom":      "value",
		}

		notifier := NewWebhookNotifier(url, headers)

		require.NotNil(t, notifier)
		assert.Equal(t, url, notifier.url)
		assert.Equal(t, headers, notifier.headers)
		assert.NotNil(t, notifier.client)
	})

	t.Run("creates notifier with nil headers", func(t *testing.T) {
		url := "https://example.com/webhook"
		notifier := NewWebhookNotifier(url, nil)

		require.NotNil(t, notifier)
		assert.Equal(t, url, notifier.url)
		assert.Nil(t, notifier.headers)
	})

	t.Run("creates notifier with empty headers", func(t *testing.T) {
		url := "https://example.com/webhook"
		headers := map[string]string{}
		notifier := NewWebhookNotifier(url, headers)

		require.NotNil(t, notifier)
		assert.Empty(t, notifier.headers)
	})
}

func TestWebhookNotifier_Name(t *testing.T) {
	t.Run("returns webhook", func(t *testing.T) {
		notifier := NewWebhookNotifier("https://example.com", nil)
		assert.Equal(t, "webhook", notifier.Name())
	})
}

func TestWebhookNotifier_Send(t *testing.T) {
	t.Run("creates correct payload", func(t *testing.T) {
		var receivedPayload WebhookPayload
		var receivedHeaders http.Header
		var mu sync.Mutex

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			defer mu.Unlock()

			receivedHeaders = r.Header.Clone()

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read body", http.StatusInternalServerError)
				return
			}

			if err := json.Unmarshal(body, &receivedPayload); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		notifier := NewWebhookNotifier(server.URL, nil)

		timestamp := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
		event := Event{
			Type:      EventDeploySucceeded,
			Project:   "myapp",
			Service:   "api",
			Status:    "healthy",
			Message:   "original message",
			Timestamp: timestamp,
			Details: map[string]string{
				"version": "1.2.3",
				"sha":     "abc123",
			},
		}

		err := notifier.Send(context.Background(), event)
		require.NoError(t, err)

		// Verify payload
		assert.Equal(t, "deploy_succeeded", receivedPayload.Type)
		assert.Equal(t, "myapp", receivedPayload.Project)
		assert.Equal(t, "api", receivedPayload.Service)
		assert.Equal(t, "healthy", receivedPayload.Status)
		assert.Equal(t, "2024-06-15T14:30:00Z", receivedPayload.Timestamp)
		assert.Equal(t, "1.2.3", receivedPayload.Details["version"])
		assert.Equal(t, "abc123", receivedPayload.Details["sha"])

		// Message should be formatted, not the original
		assert.Contains(t, receivedPayload.Message, "Deployment succeeded")

		// Verify Content-Type header
		assert.Equal(t, "application/json", receivedHeaders.Get("Content-Type"))
	})

	t.Run("includes custom headers", func(t *testing.T) {
		var receivedHeaders http.Header
		var mu sync.Mutex

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			receivedHeaders = r.Header.Clone()
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		headers := map[string]string{
			"Authorization":  "Bearer my-secret-token",
			"X-Custom-Header": "custom-value",
			"X-Request-ID":   "req-12345",
		}

		notifier := NewWebhookNotifier(server.URL, headers)

		event := Event{
			Type:      EventDeployStarted,
			Project:   "test",
			Timestamp: time.Now(),
		}

		err := notifier.Send(context.Background(), event)
		require.NoError(t, err)

		// Verify all custom headers are present
		assert.Equal(t, "Bearer my-secret-token", receivedHeaders.Get("Authorization"))
		assert.Equal(t, "custom-value", receivedHeaders.Get("X-Custom-Header"))
		assert.Equal(t, "req-12345", receivedHeaders.Get("X-Request-ID"))

		// Content-Type should still be set
		assert.Equal(t, "application/json", receivedHeaders.Get("Content-Type"))
	})

	t.Run("handles successful response codes", func(t *testing.T) {
		successCodes := []int{200, 201, 202, 204}

		for _, code := range successCodes {
			t.Run(http.StatusText(code), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(code)
				}))
				defer server.Close()

				notifier := NewWebhookNotifier(server.URL, nil)
				event := Event{
					Type:      EventDeployStarted,
					Project:   "test",
					Timestamp: time.Now(),
				}

				err := notifier.Send(context.Background(), event)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("returns error on client error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer server.Close()

		notifier := NewWebhookNotifier(server.URL, nil)
		event := Event{
			Type:      EventDeployStarted,
			Project:   "test",
			Timestamp: time.Now(),
		}

		err := notifier.Send(context.Background(), event)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})

	t.Run("uses POST method", func(t *testing.T) {
		var receivedMethod string
		var mu sync.Mutex

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			receivedMethod = r.Method
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		notifier := NewWebhookNotifier(server.URL, nil)
		event := Event{
			Type:      EventDeployStarted,
			Project:   "test",
			Timestamp: time.Now(),
		}

		err := notifier.Send(context.Background(), event)
		require.NoError(t, err)
		assert.Equal(t, http.MethodPost, receivedMethod)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		// Create a context that's already cancelled
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Use a non-routable address - no server needed since context is cancelled
		notifier := NewWebhookNotifier("http://192.0.2.1:12345/test", nil)
		event := Event{
			Type:      EventDeployStarted,
			Project:   "test",
			Timestamp: time.Now(),
		}

		err := notifier.Send(ctx, event)
		require.Error(t, err)
		// Error should be about context cancellation
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("handles empty details", func(t *testing.T) {
		var receivedPayload WebhookPayload
		var mu sync.Mutex

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			defer mu.Unlock()

			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedPayload)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		notifier := NewWebhookNotifier(server.URL, nil)
		event := Event{
			Type:      EventDeployStarted,
			Project:   "test",
			Timestamp: time.Now(),
			// Details is nil
		}

		err := notifier.Send(context.Background(), event)
		require.NoError(t, err)
		assert.Nil(t, receivedPayload.Details)
	})

	t.Run("handles empty service", func(t *testing.T) {
		var receivedPayload WebhookPayload
		var mu sync.Mutex

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			defer mu.Unlock()

			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedPayload)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		notifier := NewWebhookNotifier(server.URL, nil)
		event := Event{
			Type:      EventDeployStarted,
			Project:   "test",
			Timestamp: time.Now(),
			// Service is empty
		}

		err := notifier.Send(context.Background(), event)
		require.NoError(t, err)
		assert.Empty(t, receivedPayload.Service)
	})
}

func TestWebhookNotifier_Close(t *testing.T) {
	t.Run("does not error", func(t *testing.T) {
		notifier := NewWebhookNotifier("https://example.com/webhook", nil)
		err := notifier.Close()
		assert.NoError(t, err)
	})

	t.Run("can be called multiple times", func(t *testing.T) {
		notifier := NewWebhookNotifier("https://example.com/webhook", nil)

		err1 := notifier.Close()
		err2 := notifier.Close()
		err3 := notifier.Close()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NoError(t, err3)
	})
}

// TestWebhookNotifier_ImplementsInterface verifies the interface contract.
func TestWebhookNotifier_ImplementsInterface(t *testing.T) {
	var _ Notifier = (*WebhookNotifier)(nil)
}
