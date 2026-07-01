// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ecom-engine/internal/core/inventory/domain"
	"ecom-engine/internal/events"
)

type mockHistoryRepo struct {
	mu      sync.RWMutex
	history []*domain.StockHistory
	err     error
}

func (m *mockHistoryRepo) GetHistory(_ context.Context, _ string, _ int, _ int) ([]*domain.StockHistory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.err != nil {
		return nil, m.err
	}
	result := make([]*domain.StockHistory, len(m.history))
	for i, h := range m.history {
		if h != nil {
			hCopy := *h
			result[i] = &hCopy
		}
	}
	return result, nil
}

func (m *mockHistoryRepo) CreateHistory(_ context.Context, h *domain.StockHistory) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	if h != nil {
		hCopy := *h
		m.history = append(m.history, &hCopy)
	} else {
		m.history = append(m.history, nil)
	}
	return nil
}

type mockEventBus struct {
	subscribedTopic string
	handler         events.EventHandler
}

func (m *mockEventBus) Subscribe(topic string, handler events.EventHandler) {
	m.subscribedTopic = topic
	m.handler = handler
}

func (m *mockEventBus) Publish(_ events.Event) {}
func (m *mockEventBus) Close() error           { return nil }

func TestService_GetHistory(t *testing.T) {
	t.Parallel()

	t.Run("successful retrieval", func(t *testing.T) {
		expected := []*domain.StockHistory{
			{ID: "hist_1", VariantID: "var-1", NewQuantity: 10},
		}
		repo := &mockHistoryRepo{history: expected}
		bus := &mockEventBus{}
		svc := NewService(repo, bus)

		result, err := svc.GetHistory(context.Background(), "var-1", 20, 0)
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
	})
}

func TestService_RecordHistory(t *testing.T) {
	t.Parallel()

	t.Run("records successfully with auto ID and time", func(t *testing.T) {
		repo := &mockHistoryRepo{}
		bus := &mockEventBus{}
		svc := NewService(repo, bus)

		h := &domain.StockHistory{
			VariantID:  "var-1",
			LocationID: "default",
		}
		err := svc.RecordHistory(context.Background(), h)
		assert.NoError(t, err)
		assert.NotEmpty(t, h.ID)
		assert.False(t, h.ChangedAt.IsZero())
		assert.Len(t, repo.history, 1)
	})
}

func TestService_HandleStockChanged(t *testing.T) {
	t.Parallel()

	t.Run("handles event successfully", func(t *testing.T) {
		repo := &mockHistoryRepo{}
		bus := &mockEventBus{}
		_ = NewService(repo, bus)

		// Assert subscription registered on creation
		assert.Equal(t, events.InventoryStockChangedTopic, bus.subscribedTopic)
		assert.NotNil(t, bus.handler)

		// Create mock Event and call handler
		payload := &events.StockChangedPayload{
			VariantID:    "var-1",
			LocationID:   "loc-a",
			OldQuantity:  5,
			NewQuantity:  10,
			ChangeReason: "Restock",
			ChangedBy:    "admin",
		}
		event := events.Event{
			ID:        "evt_1",
			Topic:     events.InventoryStockChangedTopic,
			Timestamp: time.Now(),
			Payload:   payload,
		}

		require.NoError(t, bus.handler(event))

		// Verify recorded in repository
		require.Len(t, repo.history, 1)
		record := repo.history[0]
		assert.Equal(t, "var-1", record.VariantID)
		assert.Equal(t, "loc-a", record.LocationID)
		assert.Equal(t, 5, record.OldQuantity)
		assert.Equal(t, 10, record.NewQuantity)
		assert.Equal(t, "Restock", record.ChangeReason)
		assert.Equal(t, "admin", record.ChangedBy)
	})
}
