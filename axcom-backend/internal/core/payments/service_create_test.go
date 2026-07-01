// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package payments

import (
	"context"
	"ecom-engine/internal/events"
	modulespayments "ecom-engine/internal/modules/payments"
	"errors"
	"testing"
)

func TestCreatePaymentIntent(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (*MockRepository, *MockOrderFetcher, *MockPaymentProvider, Service) {
		repo := &MockRepository{payments: make(map[string]*Payment)}
		fetcher := &MockOrderFetcher{amount: 99.99, status: "pending"}
		provider := &MockPaymentProvider{intentID: "intent_123", status: "pending", redirectURL: "http://redirect.com"}
		bus := events.NewLocalEventBus()

		providers := map[string]modulespayments.PaymentProvider{
			"mock": provider,
		}

		service, err := NewPaymentService(repo, fetcher, providers, "mock", bus, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		return repo, fetcher, provider, service
	}

	t.Run("successful intent creation", func(t *testing.T) {
		t.Parallel()
		_, _, _, service := setup(t)

		pmt, err := service.CreatePaymentIntent(context.Background(), "order_1", "customer_1", "mock", "USD", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if pmt.OrderID != "order_1" {
			t.Errorf("expected OrderID to be order_1, got %s", pmt.OrderID)
		}
		if pmt.Amount != 99.99 {
			t.Errorf("expected Amount to be 99.99, got %.2f", pmt.Amount)
		}
		if pmt.ProviderIntentID != "intent_123" {
			t.Errorf("expected ProviderIntentID to be intent_123, got %s", pmt.ProviderIntentID)
		}
		if pmt.Status != StatusPending {
			t.Errorf("expected Status to be pending, got %s", pmt.Status)
		}
	})

	t.Run("order not pending", func(t *testing.T) {
		t.Parallel()
		_, fetcher, _, service := setup(t)
		fetcher.SetOrder(99.99, "paid", nil)

		_, err := service.CreatePaymentIntent(context.Background(), "order_2", "customer_1", "mock", "USD", "")
		if !errors.Is(err, ErrInvalidOrderStatus) {
			t.Errorf("expected ErrInvalidOrderStatus, got %v", err)
		}
	})

	t.Run("order not found", func(t *testing.T) {
		t.Parallel()
		_, fetcher, _, service := setup(t)
		fetcher.SetOrder(0, "", errors.New("not found"))

		_, err := service.CreatePaymentIntent(context.Background(), "order_3", "customer_1", "mock", "USD", "")
		if !errors.Is(err, ErrOrderNotFound) {
			t.Errorf("expected ErrOrderNotFound, got %v", err)
		}
	})

	t.Run("provider not found", func(t *testing.T) {
		t.Parallel()
		_, _, _, service := setup(t)

		_, err := service.CreatePaymentIntent(context.Background(), "order_1", "customer_1", "unknown_provider", "USD", "")
		if !errors.Is(err, ErrProviderNotFound) {
			t.Errorf("expected ErrProviderNotFound, got %v", err)
		}
	})

	t.Run("provider error", func(t *testing.T) {
		t.Parallel()
		_, _, provider, service := setup(t)
		provider.SetError(errors.New("provider api down"))

		_, err := service.CreatePaymentIntent(context.Background(), "order_1", "customer_1", "mock", "USD", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("idempotency returns existing pending or succeeded payment", func(t *testing.T) {
		t.Parallel()
		repo, _, _, service := setup(t)

		pmt1, err := service.CreatePaymentIntent(context.Background(), "order_100", "customer_1", "mock", "USD", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		pmt2, err := service.CreatePaymentIntent(context.Background(), "order_100", "customer_1", "mock", "USD", "")
		if err != nil {
			t.Fatalf("unexpected error on second call: %v", err)
		}

		if pmt1.ID != pmt2.ID {
			t.Errorf("expected same payment entity, got distinct IDs: %s and %s", pmt1.ID, pmt2.ID)
		}

		// Also check if status is succeeded
		pmt1.Status = StatusSucceeded
		_ = repo.Update(context.Background(), pmt1)

		pmt3, err := service.CreatePaymentIntent(context.Background(), "order_100", "customer_1", "mock", "USD", "")
		if err != nil {
			t.Fatalf("unexpected error on third call: %v", err)
		}
		if pmt3.Status != StatusSucceeded {
			t.Errorf("expected succeeded status from idempotency cache")
		}
	})

	t.Run("orphaned provider intent when repository create fails", func(t *testing.T) {
		t.Parallel()
		repo, _, _, service := setup(t)
		repo.createErr = errors.New("database disk full")

		_, err := service.CreatePaymentIntent(context.Background(), "order_200", "customer_1", "mock", "USD", "")
		if !errors.Is(err, ErrOrphanedProviderIntent) {
			t.Errorf("expected ErrOrphanedProviderIntent error, got %v", err)
		}
	})

	t.Run("invalid input validation", func(t *testing.T) {
		t.Parallel()
		_, _, _, service := setup(t)

		_, err := service.CreatePaymentIntent(context.Background(), "", "customer_1", "mock", "USD", "")
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("custom client-supplied idempotency key is stored", func(t *testing.T) {
		t.Parallel()
		_, _, _, service := setup(t)

		pmt, err := service.CreatePaymentIntent(context.Background(), "order_abc", "customer_1", "mock", "USD", "custom_idempotency_123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if pmt.IdempotencyKey != "custom_idempotency_123" {
			t.Errorf("expected IdempotencyKey to be custom_idempotency_123, got %s", pmt.IdempotencyKey)
		}
	})
}
