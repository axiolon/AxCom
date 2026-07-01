// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package payments

import (
	"context"
)

// MockPaymentsService is a flexible function-field mock of Service
type MockPaymentsService struct {
	CreatePaymentIntentFunc  func(ctx context.Context, orderID string, customerID string, providerName string, currency string, idempotencyKey string) (*Payment, error)
	ConfirmPaymentFunc       func(ctx context.Context, providerName string, intentID string) (*Payment, error)
	RefundPaymentFunc        func(ctx context.Context, orderID string, amount *float64) (*Payment, error)
	GetPaymentByOrderIDFunc  func(ctx context.Context, orderID string) (*Payment, error)
	GetPaymentByIDFunc       func(ctx context.Context, id string) (*Payment, error)
	ListAllPaymentsFunc      func(ctx context.Context, limit, offset int) ([]Payment, error)
	ListCustomerPaymentsFunc func(ctx context.Context, customerID string, limit, offset int) ([]Payment, error)
}

// CreatePaymentIntent implements Service
func (m *MockPaymentsService) CreatePaymentIntent(ctx context.Context, orderID string, customerID string, providerName string, currency string, idempotencyKey string) (*Payment, error) {
	if m.CreatePaymentIntentFunc != nil {
		return m.CreatePaymentIntentFunc(ctx, orderID, customerID, providerName, currency, idempotencyKey)
	}
	return nil, nil
}

// ConfirmPayment implements Service
func (m *MockPaymentsService) ConfirmPayment(ctx context.Context, providerName string, intentID string) (*Payment, error) {
	if m.ConfirmPaymentFunc != nil {
		return m.ConfirmPaymentFunc(ctx, providerName, intentID)
	}
	return nil, nil
}

// RefundPayment implements Service
func (m *MockPaymentsService) RefundPayment(ctx context.Context, orderID string, amount *float64) (*Payment, error) {
	if m.RefundPaymentFunc != nil {
		return m.RefundPaymentFunc(ctx, orderID, amount)
	}
	return nil, nil
}

// GetPaymentByOrderID implements Service
func (m *MockPaymentsService) GetPaymentByOrderID(ctx context.Context, orderID string) (*Payment, error) {
	if m.GetPaymentByOrderIDFunc != nil {
		return m.GetPaymentByOrderIDFunc(ctx, orderID)
	}
	return nil, nil
}

// GetPaymentByID implements Service
func (m *MockPaymentsService) GetPaymentByID(ctx context.Context, id string) (*Payment, error) {
	if m.GetPaymentByIDFunc != nil {
		return m.GetPaymentByIDFunc(ctx, id)
	}
	return nil, nil
}

// ListAllPayments implements Service
func (m *MockPaymentsService) ListAllPayments(ctx context.Context, limit, offset int) ([]Payment, error) {
	if m.ListAllPaymentsFunc != nil {
		return m.ListAllPaymentsFunc(ctx, limit, offset)
	}
	return nil, nil
}

// ListCustomerPayments implements Service
func (m *MockPaymentsService) ListCustomerPayments(ctx context.Context, customerID string, limit, offset int) ([]Payment, error) {
	if m.ListCustomerPaymentsFunc != nil {
		return m.ListCustomerPaymentsFunc(ctx, customerID, limit, offset)
	}
	return nil, nil
}
