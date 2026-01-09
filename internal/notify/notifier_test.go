package notify

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockNotifier is a test double for the Notifier interface.
type mockNotifier struct {
	name     string
	sendErr  error
	closeErr error
	sent     []Event
	mu       sync.Mutex
}

func (m *mockNotifier) Name() string { return m.name }

func (m *mockNotifier) Send(ctx context.Context, event Event) error {
	m.mu.Lock()
	m.sent = append(m.sent, event)
	m.mu.Unlock()
	return m.sendErr
}

func (m *mockNotifier) Close() error { return m.closeErr }

func (m *mockNotifier) sentEvents() []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Event{}, m.sent...)
}

// =============================================================================
// Manager Tests
// =============================================================================

func TestNewManager(t *testing.T) {
	t.Run("creates empty manager", func(t *testing.T) {
		m := NewManager()
		require.NotNil(t, m)
		assert.Equal(t, 0, m.Count())
	})
}

func TestManager_Register(t *testing.T) {
	t.Run("adds notifiers", func(t *testing.T) {
		m := NewManager()

		m.Register(&mockNotifier{name: "mock1"})
		assert.Equal(t, 1, m.Count())

		m.Register(&mockNotifier{name: "mock2"})
		assert.Equal(t, 2, m.Count())

		m.Register(&mockNotifier{name: "mock3"})
		assert.Equal(t, 3, m.Count())
	})
}

func TestManager_Count(t *testing.T) {
	tests := []struct {
		name     string
		numRegs  int
		expected int
	}{
		{"zero notifiers", 0, 0},
		{"one notifier", 1, 1},
		{"multiple notifiers", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager()
			for i := 0; i < tt.numRegs; i++ {
				m.Register(&mockNotifier{name: "mock"})
			}
			assert.Equal(t, tt.expected, m.Count())
		})
	}
}

func TestManager_Notify(t *testing.T) {
	t.Run("sends to all notifiers concurrently", func(t *testing.T) {
		m := NewManager()
		mock1 := &mockNotifier{name: "mock1"}
		mock2 := &mockNotifier{name: "mock2"}
		mock3 := &mockNotifier{name: "mock3"}

		m.Register(mock1)
		m.Register(mock2)
		m.Register(mock3)

		event := Event{
			Type:      EventDeployStarted,
			Project:   "test-project",
			Message:   "test message",
			Timestamp: time.Now(),
		}

		err := m.Notify(context.Background(), event)
		require.NoError(t, err)

		// All notifiers should have received the event
		assert.Len(t, mock1.sentEvents(), 1)
		assert.Len(t, mock2.sentEvents(), 1)
		assert.Len(t, mock3.sentEvents(), 1)

		// Verify event content
		assert.Equal(t, event.Type, mock1.sentEvents()[0].Type)
		assert.Equal(t, event.Project, mock1.sentEvents()[0].Project)
	})

	t.Run("sets timestamp if zero", func(t *testing.T) {
		m := NewManager()
		mock := &mockNotifier{name: "mock"}
		m.Register(mock)

		event := Event{
			Type:    EventDeploySucceeded,
			Project: "test-project",
			// Timestamp is zero
		}

		beforeNotify := time.Now()
		err := m.Notify(context.Background(), event)
		afterNotify := time.Now()

		require.NoError(t, err)
		require.Len(t, mock.sentEvents(), 1)

		sentEvent := mock.sentEvents()[0]
		assert.False(t, sentEvent.Timestamp.IsZero(), "timestamp should be set")
		assert.True(t, sentEvent.Timestamp.After(beforeNotify) || sentEvent.Timestamp.Equal(beforeNotify))
		assert.True(t, sentEvent.Timestamp.Before(afterNotify) || sentEvent.Timestamp.Equal(afterNotify))
	})

	t.Run("preserves existing timestamp", func(t *testing.T) {
		m := NewManager()
		mock := &mockNotifier{name: "mock"}
		m.Register(mock)

		fixedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		event := Event{
			Type:      EventDeploySucceeded,
			Project:   "test-project",
			Timestamp: fixedTime,
		}

		err := m.Notify(context.Background(), event)
		require.NoError(t, err)

		sentEvent := mock.sentEvents()[0]
		assert.Equal(t, fixedTime, sentEvent.Timestamp)
	})

	t.Run("collects errors from failed notifiers", func(t *testing.T) {
		m := NewManager()
		mock1 := &mockNotifier{name: "mock1", sendErr: errors.New("send failed")}
		mock2 := &mockNotifier{name: "mock2"} // succeeds
		mock3 := &mockNotifier{name: "mock3", sendErr: errors.New("another failure")}

		m.Register(mock1)
		m.Register(mock2)
		m.Register(mock3)

		event := Event{
			Type:      EventDeployFailed,
			Project:   "test-project",
			Timestamp: time.Now(),
		}

		err := m.Notify(context.Background(), event)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "notification errors")
		assert.Contains(t, err.Error(), "mock1")
		assert.Contains(t, err.Error(), "mock3")
	})

	t.Run("returns nil when no notifiers registered", func(t *testing.T) {
		m := NewManager()
		event := Event{
			Type:    EventDeployStarted,
			Project: "test-project",
		}

		err := m.Notify(context.Background(), event)
		assert.NoError(t, err)
	})
}

func TestManager_Close(t *testing.T) {
	t.Run("closes all notifiers", func(t *testing.T) {
		m := NewManager()
		mock1 := &mockNotifier{name: "mock1"}
		mock2 := &mockNotifier{name: "mock2"}

		m.Register(mock1)
		m.Register(mock2)

		err := m.Close()
		assert.NoError(t, err)
	})

	t.Run("collects close errors", func(t *testing.T) {
		m := NewManager()
		mock1 := &mockNotifier{name: "mock1", closeErr: errors.New("close failed")}
		mock2 := &mockNotifier{name: "mock2"}
		mock3 := &mockNotifier{name: "mock3", closeErr: errors.New("another close failure")}

		m.Register(mock1)
		m.Register(mock2)
		m.Register(mock3)

		err := m.Close()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "close errors")
		assert.Contains(t, err.Error(), "mock1")
		assert.Contains(t, err.Error(), "mock3")
	})

	t.Run("returns nil when no notifiers", func(t *testing.T) {
		m := NewManager()
		err := m.Close()
		assert.NoError(t, err)
	})
}

// =============================================================================
// FormatMessage Tests
// =============================================================================

func TestFormatMessage(t *testing.T) {
	tests := []struct {
		name     string
		event    Event
		expected string
	}{
		{
			name: "deploy_started",
			event: Event{
				Type:    EventDeployStarted,
				Project: "myapp",
			},
			expected: "🚀 Deployment started for myapp",
		},
		{
			name: "deploy_succeeded",
			event: Event{
				Type:    EventDeploySucceeded,
				Project: "myapp",
			},
			expected: "✅ Deployment succeeded for myapp",
		},
		{
			name: "deploy_failed",
			event: Event{
				Type:    EventDeployFailed,
				Project: "myapp",
				Message: "container failed to start",
			},
			expected: "❌ Deployment failed for myapp: container failed to start",
		},
		{
			name: "rollback",
			event: Event{
				Type:    EventRollback,
				Project: "myapp",
				Message: "reverting to v1.2.3",
			},
			expected: "⏪ Rollback for myapp: reverting to v1.2.3",
		},
		{
			name: "service_unhealthy",
			event: Event{
				Type:    EventServiceUnhealthy,
				Project: "myapp",
				Service: "api",
				Message: "health check failed",
			},
			expected: "⚠️ Service unhealthy: myapp/api - health check failed",
		},
		{
			name: "service_recovered",
			event: Event{
				Type:    EventServiceRecovered,
				Project: "myapp",
				Service: "api",
			},
			expected: "💚 Service recovered: myapp/api",
		},
		{
			name: "service_down",
			event: Event{
				Type:    EventServiceDown,
				Project: "myapp",
				Service: "worker",
			},
			expected: "🔴 Service down: myapp/worker",
		},
		{
			name: "service_up",
			event: Event{
				Type:    EventServiceUp,
				Project: "myapp",
				Service: "db",
			},
			expected: "🟢 Service up: myapp/db",
		},
		{
			name: "unknown event type",
			event: Event{
				Type:    EventType("custom_event"),
				Project: "myapp",
				Message: "something happened",
			},
			expected: "[custom_event] myapp: something happened",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatMessage(tt.event)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// GetEventTitle Tests
// =============================================================================

func TestGetEventTitle(t *testing.T) {
	tests := []struct {
		name      string
		eventType EventType
		expected  string
	}{
		{"deploy_started", EventDeployStarted, "🚀 Deployment Started"},
		{"deploy_succeeded", EventDeploySucceeded, "✅ Deployment Succeeded"},
		{"deploy_failed", EventDeployFailed, "❌ Deployment Failed"},
		{"rollback", EventRollback, "⏪ Rollback"},
		{"service_unhealthy", EventServiceUnhealthy, "⚠️ Service Unhealthy"},
		{"service_recovered", EventServiceRecovered, "💚 Service Recovered"},
		{"service_down", EventServiceDown, "🔴 Service Down"},
		{"service_up", EventServiceUp, "🟢 Service Up"},
		{"unknown", EventType("unknown_type"), "unknown_type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := Event{Type: tt.eventType}
			result := GetEventTitle(event)
			assert.Equal(t, tt.expected, result)
		})
	}
}
