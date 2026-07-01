// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package payments

import (
	"context"
	"ecom-engine/internal/events"
	modulespayments "ecom-engine/internal/modules/payments"
	"errors"
	"testing"
	"time"
)

func TestConfirmPayment(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (*MockRepository, *MockPaymentProvider, Service) {
		repo := &MockRepository{payments: make(map[string]*Payment)}
		fetcher := &MockOrderFetcher{amount: 99.99, status: "pending"}
		provider := &MockPaymentProvider{intentID: "intent_123", status: "pending"}
		bus := events.NewLocalEventBus()

		providers := map[string]modulespayments.PaymentProvider{
			"mock": provider,
		}

		service, err := NewPaymentService(repo, fetcher, providers, "mock", bus, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		// Seed a pending payment
		payment := &Payment{
			ID:               "pmt_1",
			OrderID:          "order_1",
			CustomerID:       "customer_1",
			Amount:           99.99,
			Currency:         "USD",
			Provider:         "mock",
			ProviderIntentID: "intent_123",
			Status:           StatusPending,
			CreatedAt:        time.Now(),
		}
		_ = repo.Create(context.Background(), payment)

		return repo, provider, service
	}

	t.Run("confirm success", func(t *testing.T) {
		t.Parallel()
		_, _, service := setup(t)

		pmt, err := service.ConfirmPayment(context.Background(), "mock", "intent_123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if pmt.Status != StatusSucceeded {
			t.Errorf("expected status to be succeeded, got %s", pmt.Status)
		}
	})

	t.Run("confirm fail", func(t *testing.T) {
		t.Parallel()
		_, provider, service := setup(t)
		provider.SetError(errors.New("gateway rejected payment"))

		pmt, err := service.ConfirmPayment(context.Background(), "mock", "intent_123")
		if err == nil {
			t.Fatal("expected error confirming failed intent, got nil")
		}

		// Verify status is marked failed in repo
		if pmt != nil {
			t.Errorf("expected return payment to be nil on provider error")
		}
	})

	t.Run("payment not found", func(t *testing.T) {
		t.Parallel()
		_, _, service := setup(t)

		_, err := service.ConfirmPayment(context.Background(), "mock", "unknown_intent")
		if !errors.Is(err, ErrPaymentNotFound) {
			t.Errorf("expected ErrPaymentNotFound, got %v", err)
		}
	})

	t.Run("provider not found", func(t *testing.T) {
		t.Parallel()
		repo, _, service := setup(t)

		// Seed a payment with "unknown_provider" so it is found in repo first
		payment := &Payment{
			ID:               "pmt_unknown_prov",
			OrderID:          "order_unknown_prov",
			CustomerID:       "customer_1",
			Amount:           99.99,
			Currency:         "USD",
			Provider:         "unknown_provider",
			ProviderIntentID: "intent_unknown",
			Status:           StatusPending,
			CreatedAt:        time.Now(),
		}
		_ = repo.Create(context.Background(), payment)

		_, err := service.ConfirmPayment(context.Background(), "unknown_provider", "intent_unknown")
		if !errors.Is(err, ErrProviderNotFound) {
			t.Errorf("expected ErrProviderNotFound, got %v", err)
		}
	})

	t.Run("already finalized succeeded", func(t *testing.T) {
		t.Parallel()
		repo, _, service := setup(t)

		// Seed a succeeded payment
		payment := &Payment{
			ID:               "pmt_succeeded",
			OrderID:          "order_succeeded",
			CustomerID:       "customer_1",
			Amount:           99.99,
			Currency:         "USD",
			Provider:         "mock",
			ProviderIntentID: "intent_succeeded",
			Status:           StatusSucceeded,
			CreatedAt:        time.Now(),
		}
		_ = repo.Create(context.Background(), payment)

		pmt, err := service.ConfirmPayment(context.Background(), "mock", "intent_succeeded")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pmt.Status != StatusSucceeded {
			t.Errorf("expected status to remain succeeded, got %s", pmt.Status)
		}
	})

	t.Run("provider confirm error", func(t *testing.T) {
		t.Parallel()
		_, provider, service := setup(t)
		provider.SetError(errors.New("failed confirming on provider gateway"))

		_, err := service.ConfirmPayment(context.Background(), "mock", "intent_123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("payment intent expired check (> 24 hours)", func(t *testing.T) {
		t.Parallel()
		repo, _, service := setup(t)

		// Seed an expired pending payment
		expiredPayment := &Payment{
			ID:               "pmt_expired",
			OrderID:          "order_expired",
			CustomerID:       "customer_1",
			Amount:           99.99,
			Currency:         "USD",
			Provider:         "mock",
			ProviderIntentID: "intent_expired",
			Status:           StatusPending,
			CreatedAt:        time.Now().Add(-25 * time.Hour),
		}
		_ = repo.Create(context.Background(), expiredPayment)

		_, err := service.ConfirmPayment(context.Background(), "mock", "intent_expired")
		if err == nil {
			t.Fatal("expected error confirming expired payment intent, got nil")
		}

		retrieved, _ := repo.GetByID(context.Background(), "pmt_expired")
		if retrieved.Status != StatusFailed {
			t.Errorf("expected status to be failed, got %s", retrieved.Status)
		}
	})
}
