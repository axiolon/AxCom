// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockAlertDispatcher is a mock implementation of AlertDispatcher.
type mockAlertDispatcher struct {
	dispatched []Alert
	err        error
}

func (m *mockAlertDispatcher) Dispatch(_ context.Context, alert Alert) error {
	if m.err != nil {
		return m.err
	}
	m.dispatched = append(m.dispatched, alert)
	return nil
}

func TestAlertDispatcher(t *testing.T) {
	alert := Alert{
		ID:        "alert_1",
		Type:      "low_stock",
		Message:   "Stock is low",
		VariantID: "var_1",
		CreatedAt: time.Now(),
		IsRead:    false,
	}

	t.Run("successful dispatch", func(t *testing.T) {
		dispatcher := &mockAlertDispatcher{}
		err := dispatcher.Dispatch(context.Background(), alert)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(dispatcher.dispatched) != 1 {
			t.Fatalf("expected 1 dispatched alert, got: %d", len(dispatcher.dispatched))
		}
		if dispatcher.dispatched[0].ID != alert.ID {
			t.Fatalf("expected alert ID %s, got %s", alert.ID, dispatcher.dispatched[0].ID)
		}
	})

	t.Run("dispatch failure", func(t *testing.T) {
		expectedErr := errors.New("dispatch error")
		dispatcher := &mockAlertDispatcher{err: expectedErr}
		err := dispatcher.Dispatch(context.Background(), alert)
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected error %v, got %v", expectedErr, err)
		}
	})
}
