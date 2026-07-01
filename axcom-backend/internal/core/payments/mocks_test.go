// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package payments

import (
	"context"
	modulespayments "ecom-engine/internal/modules/payments"
	"sync"
)

// MockRepository implements payments.Repository with thread safety
type MockRepository struct {
	mu            sync.RWMutex
	payments      map[string]*Payment
	createErr     error
	listAllErr    error
	getByOrderErr error
}

func (m *MockRepository) Create(_ context.Context, p *Payment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr != nil {
		return m.createErr
	}
	// Check duplicates based on IdempotencyKey/OrderID
	for _, existing := range m.payments {
		if existing.OrderID == p.OrderID {
			return ErrDuplicatePayment
		}
	}
	m.payments[p.ID] = p
	return nil
}

func (m *MockRepository) GetByID(_ context.Context, id string) (*Payment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, exists := m.payments[id]
	if !exists {
		return nil, ErrNotFound
	}
	return p, nil
}

func (m *MockRepository) GetByOrderID(_ context.Context, orderID string) (*Payment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getByOrderErr != nil {
		return nil, m.getByOrderErr
	}
	for _, p := range m.payments {
		if p.OrderID == orderID {
			return p, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MockRepository) GetByProviderIntentID(_ context.Context, provider string, intentID string) (*Payment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.payments {
		if p.Provider == provider && p.ProviderIntentID == intentID {
			return p, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MockRepository) Update(_ context.Context, p *Payment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.payments[p.ID] = p
	return nil
}

func (m *MockRepository) ListAll(_ context.Context, limit, offset int) ([]Payment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.listAllErr != nil {
		return nil, m.listAllErr
	}
	var list []Payment
	for _, p := range m.payments {
		list = append(list, *p)
	}

	// Apply paging simulation
	if offset > len(list) {
		return []Payment{}, nil
	}
	end := offset + limit
	if end > len(list) {
		end = len(list)
	}
	return list[offset:end], nil
}

func (m *MockRepository) ListByCustomerID(_ context.Context, customerID string, limit, offset int) ([]Payment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.listAllErr != nil {
		return nil, m.listAllErr
	}
	var list []Payment
	for _, p := range m.payments {
		if p.CustomerID == customerID {
			list = append(list, *p)
		}
	}

	// Apply paging simulation
	if offset > len(list) {
		return []Payment{}, nil
	}
	end := offset + limit
	if end > len(list) {
		end = len(list)
	}
	return list[offset:end], nil
}

// MockOrderFetcher implements OrderFetcher with thread safety
type MockOrderFetcher struct {
	mu     sync.RWMutex
	amount float64
	status string
	err    error
}

func (m *MockOrderFetcher) GetOrderAmountAndStatus(_ context.Context, _ string) (float64, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.amount, m.status, m.err
}

func (m *MockOrderFetcher) SetOrder(amount float64, status string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.amount = amount
	m.status = status
	m.err = err
}

// MockPaymentProvider implements modulespayments.PaymentProvider with thread safety
type MockPaymentProvider struct {
	mu          sync.RWMutex
	intentID    string
	status      string
	redirectURL string
	err         error
}

func (m *MockPaymentProvider) CreateIntent(_ context.Context, amount float64, currency string) (*modulespayments.PaymentIntent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.err != nil {
		return nil, m.err
	}
	return &modulespayments.PaymentIntent{
		ID:          m.intentID,
		Amount:      amount,
		Currency:    currency,
		Status:      m.status,
		RedirectURL: m.redirectURL,
	}, nil
}

func (m *MockPaymentProvider) ConfirmIntent(_ context.Context, _ string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.err
}

func (m *MockPaymentProvider) RefundIntent(_ context.Context, _ string, _ float64) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.err
}

func (m *MockPaymentProvider) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}
