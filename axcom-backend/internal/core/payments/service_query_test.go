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

func TestGetPaymentByOrderID(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (*MockRepository, Service) {
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
		return repo, service
	}

	t.Run("get payment success", func(t *testing.T) {
		t.Parallel()
		repo, service := setup(t)

		payment := &Payment{
			ID:      "pmt_get",
			OrderID: "order_get",
			Status:  StatusPending,
		}
		err := repo.Create(context.Background(), payment)
		if err != nil {
			t.Fatal(err)
		}

		pmt, err := service.GetPaymentByOrderID(context.Background(), "order_get")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pmt.ID != "pmt_get" {
			t.Errorf("expected ID to be pmt_get, got %s", pmt.ID)
		}
	})

	t.Run("get payment not found", func(t *testing.T) {
		t.Parallel()
		_, service := setup(t)

		_, err := service.GetPaymentByOrderID(context.Background(), "unknown_order")
		if !errors.Is(err, ErrPaymentNotFound) {
			t.Errorf("expected ErrPaymentNotFound, got %v", err)
		}
	})

	t.Run("get payment database error is preserved", func(t *testing.T) {
		t.Parallel()
		repo, service := setup(t)
		repo.getByOrderErr = errors.New("repository: connection refused")

		_, err := service.GetPaymentByOrderID(context.Background(), "order_any")
		if errors.Is(err, ErrPaymentNotFound) {
			t.Errorf("expected original repository error, got ErrPaymentNotFound")
		}
	})
}

func TestListAllPayments(t *testing.T) {
	t.Parallel()

	fetcher := &MockOrderFetcher{amount: 99.99, status: "pending"}
	provider := &MockPaymentProvider{intentID: "intent_123", status: "pending"}
	bus := events.NewLocalEventBus()

	providers := map[string]modulespayments.PaymentProvider{
		"mock": provider,
	}

	t.Run("list payments success with pagination limits", func(t *testing.T) {
		t.Parallel()
		subRepo := &MockRepository{payments: make(map[string]*Payment)}
		subService, err := NewPaymentService(subRepo, fetcher, providers, "mock", bus, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		p1 := &Payment{ID: "p1", OrderID: "o1"}
		p2 := &Payment{ID: "p2", OrderID: "o2"}
		_ = subRepo.Create(context.Background(), p1)
		_ = subRepo.Create(context.Background(), p2)

		list, err := subService.ListAllPayments(context.Background(), 1, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 1 {
			t.Errorf("expected 1 payment (limit=1), got %d", len(list))
		}

		listOffset, err := subService.ListAllPayments(context.Background(), 1, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(listOffset) != 1 {
			t.Errorf("expected 1 payment (limit=1, offset=1), got %d", len(listOffset))
		}
	})
}

func TestGetPaymentByID(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (*MockRepository, Service) {
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
		return repo, service
	}

	t.Run("get payment by id success", func(t *testing.T) {
		t.Parallel()
		repo, service := setup(t)

		payment := &Payment{
			ID:      "pmt_123",
			OrderID: "order_123",
			Status:  StatusPending,
		}
		err := repo.Create(context.Background(), payment)
		if err != nil {
			t.Fatal(err)
		}

		pmt, err := service.GetPaymentByID(context.Background(), "pmt_123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pmt.ID != "pmt_123" {
			t.Errorf("expected ID to be pmt_123, got %s", pmt.ID)
		}
	})

	t.Run("get payment by id not found", func(t *testing.T) {
		t.Parallel()
		_, service := setup(t)

		_, err := service.GetPaymentByID(context.Background(), "unknown_pmt")
		if !errors.Is(err, ErrPaymentNotFound) {
			t.Errorf("expected ErrPaymentNotFound, got %v", err)
		}
	})
}
