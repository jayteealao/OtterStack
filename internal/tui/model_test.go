package tui

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestViewConstants(t *testing.T) {
	t.Run("ViewList is 0", func(t *testing.T) {
		assert.Equal(t, View(0), ViewList)
	})

	t.Run("ViewDetail is 1", func(t *testing.T) {
		assert.Equal(t, View(1), ViewDetail)
	})
}

func TestNewModel(t *testing.T) {
	t.Run("creates model with correct initial values", func(t *testing.T) {
		ctx := context.Background()
		refreshInterval := 5 * time.Second

		// Note: store is nil for this test since we can't easily create
		// a real store without database access. Integration tests would
		// be needed for full store testing.
		m := NewModel(ctx, nil, refreshInterval)

		// Verify initial state
		assert.Equal(t, ViewList, m.currentView, "should start in list view")
		assert.Equal(t, 0, m.selectedIndex, "selected index should be 0")
		assert.Equal(t, refreshInterval, m.refreshTicker, "refresh interval should match")
		assert.False(t, m.quitting, "should not be quitting initially")
		assert.Nil(t, m.err, "should have no error initially")
		assert.Empty(t, m.projects, "should have no projects initially")
	})

	t.Run("sets up context and cancel function", func(t *testing.T) {
		ctx := context.Background()
		m := NewModel(ctx, nil, time.Second)

		// Verify context is set
		require.NotNil(t, m.ctx, "context should be set")
		require.NotNil(t, m.cancel, "cancel function should be set")

		// Verify cancel works without panic
		assert.NotPanics(t, func() {
			m.cancel()
		}, "cancel should not panic")

		// After cancellation, context should be done
		select {
		case <-m.ctx.Done():
			// Expected behavior
		default:
			t.Error("context should be done after cancel")
		}
	})

	t.Run("initializes table with correct columns", func(t *testing.T) {
		ctx := context.Background()
		m := NewModel(ctx, nil, time.Second)

		// Get table columns - the table model exposes them via View
		// We can verify the table is initialized by checking it's not zero
		view := m.table.View()
		assert.NotEmpty(t, view, "table view should not be empty")

		// Verify the expected column headers appear in the view
		// The table should have PROJECT, TYPE, REF, STATUS, SERVICES columns
		assert.Contains(t, view, "PROJECT", "table should have PROJECT column")
		assert.Contains(t, view, "TYPE", "table should have TYPE column")
		assert.Contains(t, view, "REF", "table should have REF column")
		assert.Contains(t, view, "STATUS", "table should have STATUS column")
		assert.Contains(t, view, "SERVICES", "table should have SERVICES column")
	})

	t.Run("accepts nil store", func(t *testing.T) {
		ctx := context.Background()

		// Should not panic with nil store
		assert.NotPanics(t, func() {
			NewModel(ctx, nil, time.Second)
		}, "NewModel should accept nil store")
	})

	t.Run("uses provided refresh interval", func(t *testing.T) {
		ctx := context.Background()

		intervals := []time.Duration{
			1 * time.Second,
			5 * time.Second,
			30 * time.Second,
			1 * time.Minute,
		}

		for _, interval := range intervals {
			m := NewModel(ctx, nil, interval)
			assert.Equal(t, interval, m.refreshTicker,
				"refresh ticker should be %v", interval)
		}
	})
}

func TestProjectInfo(t *testing.T) {
	t.Run("zero value is valid", func(t *testing.T) {
		var info ProjectInfo
		assert.Nil(t, info.Project)
		assert.Nil(t, info.Deployment)
		assert.Nil(t, info.Services)
		assert.Nil(t, info.Error)
	})
}

func TestKeyMap(t *testing.T) {
	// Verify keybindings are defined
	t.Run("keybindings are configured", func(t *testing.T) {
		// Up binding
		assert.True(t, keys.Up.Enabled(), "up key should be enabled")

		// Down binding
		assert.True(t, keys.Down.Enabled(), "down key should be enabled")

		// Enter binding
		assert.True(t, keys.Enter.Enabled(), "enter key should be enabled")

		// Back binding
		assert.True(t, keys.Back.Enabled(), "back key should be enabled")

		// Refresh binding
		assert.True(t, keys.Refresh.Enabled(), "refresh key should be enabled")

		// Quit binding
		assert.True(t, keys.Quit.Enabled(), "quit key should be enabled")

		// Help binding
		assert.True(t, keys.Help.Enabled(), "help key should be enabled")
	})
}
