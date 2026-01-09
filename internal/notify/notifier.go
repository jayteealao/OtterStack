// Package notify provides notification backends for OtterStack alerts.
package notify

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Event represents a notification event.
type Event struct {
	Type      EventType
	Project   string
	Service   string
	Status    string
	Message   string
	Timestamp time.Time
	Details   map[string]string
}

// EventType represents the type of notification event.
type EventType string

const (
	EventDeployStarted   EventType = "deploy_started"
	EventDeploySucceeded EventType = "deploy_succeeded"
	EventDeployFailed    EventType = "deploy_failed"
	EventRollback        EventType = "rollback"
	EventServiceUnhealthy EventType = "service_unhealthy"
	EventServiceRecovered EventType = "service_recovered"
	EventServiceDown     EventType = "service_down"
	EventServiceUp       EventType = "service_up"
)

// Notifier is the interface for notification backends.
type Notifier interface {
	// Name returns the name of the notifier.
	Name() string

	// Send sends a notification event.
	Send(ctx context.Context, event Event) error

	// Close cleans up any resources.
	Close() error
}

// Config holds configuration for a notifier.
type Config struct {
	Type    string            `json:"type"`
	Enabled bool              `json:"enabled"`
	Options map[string]string `json:"options"`
}

// Manager manages multiple notification backends.
type Manager struct {
	notifiers []Notifier
}

// NewManager creates a new notification manager.
func NewManager() *Manager {
	return &Manager{
		notifiers: make([]Notifier, 0),
	}
}

// Register adds a notifier to the manager.
func (m *Manager) Register(n Notifier) {
	m.notifiers = append(m.notifiers, n)
}

// Notify sends an event to all registered notifiers.
func (m *Manager) Notify(ctx context.Context, event Event) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	for _, n := range m.notifiers {
		wg.Add(1)
		go func(notifier Notifier) {
			defer wg.Done()
			if err := notifier.Send(ctx, event); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: %w", notifier.Name(), err))
				mu.Unlock()
			}
		}(n)
	}
	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("notification errors: %v", errs)
	}
	return nil
}

// Close closes all registered notifiers.
func (m *Manager) Close() error {
	var errs []error
	for _, n := range m.notifiers {
		if err := n.Close(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", n.Name(), err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

// Count returns the number of registered notifiers.
func (m *Manager) Count() int {
	return len(m.notifiers)
}

// FormatMessage creates a human-readable message from an event.
func FormatMessage(event Event) string {
	switch event.Type {
	case EventDeployStarted:
		return fmt.Sprintf("🚀 Deployment started for %s", event.Project)
	case EventDeploySucceeded:
		return fmt.Sprintf("✅ Deployment succeeded for %s", event.Project)
	case EventDeployFailed:
		return fmt.Sprintf("❌ Deployment failed for %s: %s", event.Project, event.Message)
	case EventRollback:
		return fmt.Sprintf("⏪ Rollback for %s: %s", event.Project, event.Message)
	case EventServiceUnhealthy:
		return fmt.Sprintf("⚠️ Service unhealthy: %s/%s - %s", event.Project, event.Service, event.Message)
	case EventServiceRecovered:
		return fmt.Sprintf("💚 Service recovered: %s/%s", event.Project, event.Service)
	case EventServiceDown:
		return fmt.Sprintf("🔴 Service down: %s/%s", event.Project, event.Service)
	case EventServiceUp:
		return fmt.Sprintf("🟢 Service up: %s/%s", event.Project, event.Service)
	default:
		return fmt.Sprintf("[%s] %s: %s", event.Type, event.Project, event.Message)
	}
}

// GetEventTitle returns a human-readable title for an event type.
func GetEventTitle(event Event) string {
	switch event.Type {
	case EventDeployStarted:
		return "🚀 Deployment Started"
	case EventDeploySucceeded:
		return "✅ Deployment Succeeded"
	case EventDeployFailed:
		return "❌ Deployment Failed"
	case EventRollback:
		return "⏪ Rollback"
	case EventServiceUnhealthy:
		return "⚠️ Service Unhealthy"
	case EventServiceRecovered:
		return "💚 Service Recovered"
	case EventServiceDown:
		return "🔴 Service Down"
	case EventServiceUp:
		return "🟢 Service Up"
	default:
		return string(event.Type)
	}
}

// retryableSend executes an HTTP request with retry logic for transient failures.
func retryableSend(ctx context.Context, client *http.Client, req *http.Request, maxRetries int) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(1<<uint(attempt-1)) * time.Second):
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Don't retry client errors (4xx), only server errors (5xx)
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: status %d", resp.StatusCode)
			continue
		}

		return resp, nil
	}
	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
