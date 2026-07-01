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

func TestRefundPayment(t *testing.T) {
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
		return repo, provider, service
	}

	t.Run("refund success", func(t *testing.T) {
		t.Parallel()
		repo, _, service := setup(t)

		payment := &Payment{
			ID:               "pmt_refund",
			OrderID:          "order_refund",
			CustomerID:       "customer_1",
			Amount:           99.99,
			Currency:         "USD",
			Provider:         "mock",
			ProviderIntentID: "intent_refund",
			Status:           StatusSucceeded,
		}
		repo.payments[payment.ID] = payment

		pmt, err := service.RefundPayment(context.Background(), "order_refund", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if pmt.Status != StatusRefunded {
			t.Errorf("expected status to be refunded, got %s", pmt.Status)
		}
		if pmt.RefundedAt == nil {
			t.Errorf("expected RefundedAt to be populated, got nil")
		}
	})

	t.Run("partial refund success", func(t *testing.T) {
		t.Parallel()
		repo, _, service := setup(t)

		payment := &Payment{
			ID:               "pmt_partial",
			OrderID:          "order_partial",
			CustomerID:       "customer_1",
			Amount:           100.00,
			Currency:         "USD",
			Provider:         "mock",
			ProviderIntentID: "intent_partial",
			Status:           StatusSucceeded,
		}
		repo.payments[payment.ID] = payment

		partialAmount := 40.00
		pmt, err := service.RefundPayment(context.Background(), "order_partial", &partialAmount)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if pmt.Status != StatusRefunded {
			t.Errorf("expected status to be refunded, got %s", pmt.Status)
		}
	})

	t.Run("partial refund invalid amount", func(t *testing.T) {
		t.Parallel()
		repo, _, service := setup(t)

		payment := &Payment{
			ID:               "pmt_partial_inv",
			OrderID:          "order_partial_inv",
			CustomerID:       "customer_1",
			Amount:           100.00,
			Currency:         "USD",
			Provider:         "mock",
			ProviderIntentID: "intent_partial_inv",
			Status:           StatusSucceeded,
		}
		repo.payments[payment.ID] = payment

		invalidAmount := 150.00
		_, err := service.RefundPayment(context.Background(), "order_partial_inv", &invalidAmount)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("payment not found", func(t *testing.T) {
		t.Parallel()
		_, _, service := setup(t)

		_, err := service.RefundPayment(context.Background(), "unknown_order", nil)
		if !errors.Is(err, ErrPaymentNotFound) {
			t.Errorf("expected ErrPaymentNotFound, got %v", err)
		}
	})

	t.Run("payment not succeeded", func(t *testing.T) {
		t.Parallel()
		repo, _, service := setup(t)

		payment := &Payment{
			ID:               "pmt_pending",
			OrderID:          "order_pending",
			CustomerID:       "customer_1",
			Amount:           99.99,
			Currency:         "USD",
			Provider:         "mock",
			ProviderIntentID: "intent_pending",
			Status:           StatusPending,
		}
		repo.payments[payment.ID] = payment

		_, err := service.RefundPayment(context.Background(), "order_pending", nil)
		if err == nil {
			t.Fatal("expected error when refunding non-succeeded payment, got nil")
		}
	})

	t.Run("provider not found for refund", func(t *testing.T) {
		t.Parallel()
		repo, _, service := setup(t)

		payment := &Payment{
			ID:               "pmt_refund",
			OrderID:          "order_refund",
			CustomerID:       "customer_1",
			Amount:           99.99,
			Currency:         "USD",
			Provider:         "unknown_provider",
			ProviderIntentID: "intent_refund",
			Status:           StatusSucceeded,
		}
		repo.payments[payment.ID] = payment

		_, err := service.RefundPayment(context.Background(), "order_refund", nil)
		if !errors.Is(err, ErrProviderNotFound) {
			t.Errorf("expected ErrProviderNotFound, got %v", err)
		}
	})

	t.Run("provider refund error results in refund_failed compensating status", func(t *testing.T) {
		t.Parallel()
		repo, provider, service := setup(t)
		provider.SetError(errors.New("refund limits exceeded on gateway"))

		payment := &Payment{
			ID:               "pmt_refund",
			OrderID:          "order_refund",
			CustomerID:       "customer_1",
			Amount:           99.99,
			Currency:         "USD",
			Provider:         "mock",
			ProviderIntentID: "intent_refund",
			Status:           StatusSucceeded,
		}
		repo.payments[payment.ID] = payment

		_, err := service.RefundPayment(context.Background(), "order_refund", nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		retrieved, _ := repo.GetByID(context.Background(), payment.ID)
		if retrieved.Status != StatusRefundFailed {
			t.Errorf("expected status to be refund_failed, got %s", retrieved.Status)
		}
	})
}

func TestOrderCancelledSubscriber(t *testing.T) {
	repo := &MockRepository{payments: make(map[string]*Payment)}
	fetcher := &MockOrderFetcher{amount: 99.99, status: "paid"}
	provider := &MockPaymentProvider{intentID: "intent_123", status: "succeeded"}
	bus := events.NewLocalEventBus() // Use real local event bus implementation

	providers := map[string]modulespayments.PaymentProvider{
		"mock": provider,
	}

	service, err := NewPaymentService(repo, fetcher, providers, "mock", bus, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = service // ensure it is registered

	payment := &Payment{
		ID:               "pmt_refund",
		OrderID:          "order_refund",
		CustomerID:       "customer_1",
		Amount:           99.99,
		Currency:         "USD",
		Provider:         "mock",
		ProviderIntentID: "intent_refund",
		Status:           StatusSucceeded,
	}
	repo.payments[payment.ID] = payment

	// Publish the OrderCancelledTopic event
	done := make(chan bool)
	bus.Subscribe(events.PaymentRefundedTopic, func(_ events.Event) error {
		done <- true
		return nil
	})

	bus.Publish(events.NewEvent(
		events.OrderCancelledTopic,
		"orders",
		events.OrderCancelledEventPayload{
			OrderID: "order_refund",
			Reason:  "canceled",
		},
	))

	select {
	case <-done:
		// Successfully triggered and finished the async refund flow
		refundedPayment, err := repo.GetByOrderID(context.Background(), "order_refund")
		if err != nil {
			t.Fatalf("failed to retrieve updated payment: %v", err)
		}
		if refundedPayment.Status != StatusRefunded {
			t.Errorf("expected payment status to be refunded, got %s", refundedPayment.Status)
		}
	case <-time.After(time.Second * 5):
		t.Fatal("timed out waiting for background refund event processing")
	}
}
