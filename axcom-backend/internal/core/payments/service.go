// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package payments

import (
	"context"
	"ecom-engine/internal/events"
	infradb "ecom-engine/internal/infra/db"
	modulespayments "ecom-engine/internal/modules/payments"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"ecom-engine/pkg/logger"

	"ecom-engine/pkg/idgen"
)

var (
	defaultTimeout = 15 * time.Second
)

func getProviderTimeout() time.Duration {
	if val := os.Getenv("PAYMENT_PROVIDER_TIMEOUT_SEC"); val != "" {
		if sec, err := strconv.Atoi(val); err == nil {
			return time.Duration(sec) * time.Second
		}
		logger.Warn("PAYMENT_PROVIDER_TIMEOUT_SEC=%q is not a valid integer; using default timeout of %v", os.Getenv("PAYMENT_PROVIDER_TIMEOUT_SEC"), defaultTimeout)
	}
	return defaultTimeout
}

var (
	ErrOrderNotFound           = errors.New("order not found")
	ErrInvalidOrderStatus      = errors.New("order is not in pending status")
	ErrPaymentNotFound         = errors.New("payment not found")
	ErrProviderNotFound        = errors.New("payment provider not found")
	ErrInvalidInput            = errors.New("invalid input parameters")
	ErrDuplicatePaymentService = errors.New("payment already initiated/succeeded for this order")
	ErrOrphanedProviderIntent  = errors.New("payment intent created but failed to save in system")
)

// OrderFetcher is a mini-interface to fetch order info without cyclic dependency.
type OrderFetcher interface {
	GetOrderAmountAndStatus(ctx context.Context, orderID string) (float64, string, error)
}

type Service interface {
	CreatePaymentIntent(ctx context.Context, orderID string, customerID string, providerName string, currency string, idempotencyKey string) (*Payment, error)
	ConfirmPayment(ctx context.Context, providerName string, intentID string) (*Payment, error)
	RefundPayment(ctx context.Context, orderID string, amount *float64) (*Payment, error)
	GetPaymentByOrderID(ctx context.Context, orderID string) (*Payment, error)
	GetPaymentByID(ctx context.Context, id string) (*Payment, error)
	ListAllPayments(ctx context.Context, limit, offset int) ([]Payment, error)
	ListCustomerPayments(ctx context.Context, customerID string, limit, offset int) ([]Payment, error)
}

type paymentService struct {
	repo         Repository
	orderFetcher OrderFetcher
	providers    map[string]modulespayments.PaymentProvider
	defaultProv  string
	eventBus     events.EventBus
	outbox       events.OutboxRepository
	txManager    infradb.TransactionManager
	timeout      time.Duration
}

func NewPaymentService(
	repo Repository,
	orderFetcher OrderFetcher,
	providers map[string]modulespayments.PaymentProvider,
	defaultProv string,
	eventBus events.EventBus,
	outbox events.OutboxRepository,
	txManager infradb.TransactionManager,
) (Service, error) {
	safeProviders := make(map[string]modulespayments.PaymentProvider)
	for k, v := range providers {
		safeProviders[k] = v
	}
	if defaultProv != "" {
		if _, exists := safeProviders[defaultProv]; !exists {
			return nil, fmt.Errorf("default payment provider %q does not exist in registered providers", defaultProv)
		}
	}
	s := &paymentService{
		repo:         repo,
		orderFetcher: orderFetcher,
		providers:    safeProviders,
		defaultProv:  defaultProv,
		eventBus:     eventBus,
		outbox:       outbox,
		txManager:    txManager,
		timeout:      getProviderTimeout(),
	}
	if eventBus != nil {
		eventBus.Subscribe(events.OrderCancelledTopic, s.handleOrderCancelled)
	}
	return s, nil
}

func (s *paymentService) CreatePaymentIntent(ctx context.Context, orderID string, customerID string, providerName string, currency string, idempotencyKey string) (*Payment, error) {
	if orderID == "" || customerID == "" {
		return nil, ErrInvalidInput
	}
	if currency == "" {
		currency = "USD"
	}

	logger.InfoCtx(ctx, "Creating payment intent for order %s, customer %s", orderID, customerID)

	// Idempotency precheck: if a payment already exists for this order, handle it.
	// - succeeded/pending: return as-is (idempotent).
	// - failed/refund_failed: fall through to allow re-attempt; the order status
	//   guard below ensures we only proceed if the order is still "pending".
	existing, err := s.repo.GetByOrderID(ctx, orderID)
	if err == nil {
		switch existing.Status {
		case StatusSucceeded, StatusPending:
			return existing, nil
		case StatusFailed, StatusRefundFailed:
			logger.InfoCtx(ctx, "Existing payment %s for order %s is in status %s; allowing re-attempt", existing.ID, orderID, existing.Status)
		}
	} else if !errors.Is(err, ErrNotFound) {
		logger.ErrorCtx(ctx, "Database failure checking existing payment for order %s: %v", orderID, err)
		return nil, fmt.Errorf("repository error: %w", err)
	}

	// Fetch order total and status using OrderFetcher
	total, status, err := s.orderFetcher.GetOrderAmountAndStatus(ctx, orderID)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to fetch order: %v", err)
		return nil, ErrOrderNotFound
	}

	if status != "pending" {
		logger.ErrorCtx(ctx, "Order %s is in %s state, cannot initiate payment", orderID, status)
		return nil, ErrInvalidOrderStatus
	}

	if providerName == "" {
		providerName = s.defaultProv
	}

	provider, exists := s.providers[providerName]
	if !exists {
		logger.ErrorCtx(ctx, "Payment provider %s not found", providerName)
		return nil, ErrProviderNotFound
	}

	// Create intent with external provider enforcing configurable timeout context
	provCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	intent, err := provider.CreateIntent(provCtx, total, currency)
	if err != nil {
		logger.ErrorCtx(ctx, "Provider failed to create intent: %v", err)
		return nil, fmt.Errorf("provider error: %w", err)
	}

	if idempotencyKey == "" {
		idempotencyKey = orderID
	}

	payID, err := idgen.Generate("pmt_")
	if err != nil {
		return nil, fmt.Errorf("failed to generate payment ID: %w", err)
	}

	payment := &Payment{
		ID:               payID,
		OrderID:          orderID,
		CustomerID:       customerID,
		Amount:           total,
		Currency:         currency,
		Provider:         providerName,
		ProviderIntentID: intent.ID,
		Status:           PaymentStatus(intent.Status),
		IdempotencyKey:   idempotencyKey,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Save to DB
	if err := s.repo.Create(ctx, payment); err != nil {
		if errors.Is(err, ErrDuplicatePayment) {
			logger.WarnCtx(ctx, "Concurrent request created duplicate payment for order %s: %v", orderID, err)
			return nil, ErrDuplicatePaymentService
		}
		// CRITICAL: provider intent was created but we can't persist it — manual reconciliation needed.
		logger.ErrorCtx(ctx, "[CRITICAL-MONEY-LOST] Payment intent created on provider %s (IntentID: %s, Amount: %.2f %s) but failed to save to database: %v. OrderID: %s, CustomerID: %s",
			providerName, intent.ID, total, currency, err, orderID, customerID)
		return nil, fmt.Errorf("%w: %v", ErrOrphanedProviderIntent, err)
	}

	// If the provider returned "succeeded" immediately (e.g. mock/sandbox), publish event now.
	if payment.Status == StatusSucceeded {
		s.publishPaymentSuccess(ctx, payment)
	}

	return payment, nil
}

func (s *paymentService) ConfirmPayment(ctx context.Context, providerName string, intentID string) (*Payment, error) {
	if providerName == "" || intentID == "" {
		return nil, ErrInvalidInput
	}

	logger.InfoCtx(ctx, "Confirming payment with provider %s, intent ID %s", providerName, intentID)

	payment, err := s.repo.GetByProviderIntentID(ctx, providerName, intentID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			logger.ErrorCtx(ctx, "Payment record not found for intent: %s, provider: %s", intentID, providerName)
			return nil, ErrPaymentNotFound
		}
		logger.ErrorCtx(ctx, "Database error retrieving payment: %v", err)
		return nil, fmt.Errorf("repository error: %w", err)
	}

	if payment.Status == StatusSucceeded || payment.Status == StatusRefunded {
		logger.InfoCtx(ctx, "Payment %s already in final state: %s", payment.ID, payment.Status)
		return payment, nil
	}

	if payment.Status == StatusFailed {
		logger.ErrorCtx(ctx, "Payment %s is in failed status, cannot confirm or retry", payment.ID)
		return nil, errors.New("cannot confirm failed payment")
	}

	if time.Since(payment.CreatedAt) > 24*time.Hour {
		logger.ErrorCtx(ctx, "Payment %s confirmation rejected: intent created more than 24 hours ago", payment.ID)
		payment.Status = StatusFailed
		payment.FailureReason = "payment intent expired (older than 24 hours)"
		payment.UpdatedAt = time.Now()
		if updateErr := s.repo.Update(ctx, payment); updateErr != nil {
			logger.ErrorCtx(ctx, "Failed to persist expired payment status for %s: %v", payment.ID, updateErr)
		}
		s.publishPaymentFailure(ctx, payment, payment.FailureReason)
		return nil, errors.New("cannot confirm expired payment intent")
	}

	provider, exists := s.providers[providerName]
	if !exists {
		logger.ErrorCtx(ctx, "Payment provider %s not found", providerName)
		return nil, ErrProviderNotFound
	}

	// Enforce provider timeout
	provCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	confirmErr := provider.ConfirmIntent(provCtx, intentID)
	if confirmErr != nil {
		logger.ErrorCtx(ctx, "Provider failed to confirm intent %s: %v", intentID, confirmErr)
		payment.Status = StatusFailed
		payment.FailureReason = confirmErr.Error()
		payment.UpdatedAt = time.Now()
		if updateErr := s.repo.Update(ctx, payment); updateErr != nil {
			logger.ErrorCtx(ctx, "Failed to update failed payment status in database: %v", updateErr)
			// Return both errors: the DB failure is the immediate problem, but preserve the provider error for diagnostics.
			return nil, fmt.Errorf("failed to save payment failure (db: %v; provider: %w)", updateErr, confirmErr)
		}
		s.publishPaymentFailure(ctx, payment, confirmErr.Error())
		return nil, fmt.Errorf("provider error: %w", confirmErr)
	}

	payment.Status = StatusSucceeded
	payment.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, payment); err != nil {
		logger.ErrorCtx(ctx, "Failed to update payment status in database: %v", err)
		return nil, fmt.Errorf("repository error: %w", err)
	}

	s.publishPaymentSuccess(ctx, payment)

	return payment, nil
}

func (s *paymentService) RefundPayment(ctx context.Context, orderID string, amount *float64) (*Payment, error) {
	if orderID == "" {
		return nil, ErrInvalidInput
	}

	logger.InfoCtx(ctx, "Refunding payment for order %s", orderID)

	payment, err := s.repo.GetByOrderID(ctx, orderID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrPaymentNotFound
		}
		logger.ErrorCtx(ctx, "Failed to retrieve payment for order %s: %v", orderID, err)
		return nil, fmt.Errorf("repository error: %w", err)
	}

	if payment.Status == StatusRefunded {
		logger.InfoCtx(ctx, "Payment %s already refunded", payment.ID)
		return payment, nil
	}

	if payment.Status != StatusSucceeded && payment.Status != StatusRefundFailed {
		logger.ErrorCtx(ctx, "Payment %s is in status %s, cannot refund", payment.ID, payment.Status)
		return nil, fmt.Errorf("cannot refund payment in status %s", payment.Status)
	}

	refundAmount := payment.Amount
	if amount != nil {
		if *amount <= 0 || *amount > payment.Amount {
			return nil, fmt.Errorf("%w: invalid refund amount", ErrInvalidInput)
		}
		refundAmount = *amount
	}

	provider, exists := s.providers[payment.Provider]
	if !exists {
		return nil, ErrProviderNotFound
	}

	// Optimistic DB write before calling provider to prevent double-refund on retry.
	// If the provider call fails, we compensate by rolling back to refund_failed.
	now := time.Now()
	payment.Status = StatusRefunded
	payment.UpdatedAt = now
	payment.RefundedAt = &now

	if err = s.repo.Update(ctx, payment); err != nil {
		logger.ErrorCtx(ctx, "Failed to record refund intent in DB: %v", err)
		return nil, fmt.Errorf("failed to initiate refund: %w", err)
	}

	provCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	err = provider.RefundIntent(provCtx, payment.ProviderIntentID, refundAmount)
	if err != nil {
		logger.ErrorCtx(ctx, "Provider failed to refund payment %s (order %s): %v", payment.ID, orderID, err)
		// Compensating transaction: mark as refund_failed so operators can retry.
		payment.Status = StatusRefundFailed
		payment.RefundedAt = nil
		payment.UpdatedAt = time.Now()
		if rollbackErr := s.repo.Update(ctx, payment); rollbackErr != nil {
			// Both the provider refund AND the DB rollback failed — the record is stuck as "refunded"
			// but no money was actually returned. This requires manual intervention.
			logger.ErrorCtx(ctx, "[CRITICAL-REFUND-STUCK] Failed to roll back refund status for payment %s (order %s): db_err=%v; provider_err=%v",
				payment.ID, orderID, rollbackErr, err)
		}
		return nil, fmt.Errorf("provider error: %w", err)
	}

	s.publishPaymentRefund(ctx, payment)

	return payment, nil
}

func (s *paymentService) GetPaymentByID(ctx context.Context, id string) (*Payment, error) {
	if id == "" {
		return nil, ErrInvalidInput
	}
	payment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("repository error: %w", err)
	}
	return payment, nil
}

func (s *paymentService) GetPaymentByOrderID(ctx context.Context, orderID string) (*Payment, error) {
	if orderID == "" {
		return nil, ErrInvalidInput
	}
	payment, err := s.repo.GetByOrderID(ctx, orderID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("repository error: %w", err)
	}
	return payment, nil
}

func (s *paymentService) ListAllPayments(ctx context.Context, limit, offset int) ([]Payment, error) {
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListAll(ctx, limit, offset)
}

func (s *paymentService) ListCustomerPayments(ctx context.Context, customerID string, limit, offset int) ([]Payment, error) {
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListByCustomerID(ctx, customerID, limit, offset)
}

func (s *paymentService) publishPaymentSuccess(ctx context.Context, p *Payment) {
	evt := events.NewEventFromCtx(ctx, events.PaymentSucceededTopic, "payments",
		events.PaymentEventPayload{
			OrderID:    p.OrderID,
			PaymentID:  p.ID,
			CustomerID: p.CustomerID,
			Amount:     p.Amount,
		})
	s.publishEvent(ctx, evt)
}

func (s *paymentService) publishPaymentFailure(ctx context.Context, p *Payment, errStr string) {
	if errStr == "" {
		errStr = "payment failed"
	}
	evt := events.NewEventFromCtx(ctx, events.PaymentFailedTopic, "payments",
		events.PaymentFailedEventPayload{
			OrderID:    p.OrderID,
			PaymentID:  p.ID,
			CustomerID: p.CustomerID,
			Amount:     p.Amount,
			Error:      errStr,
		})
	s.publishEvent(ctx, evt)
}

func (s *paymentService) publishPaymentRefund(ctx context.Context, p *Payment) {
	refundedTime := p.UpdatedAt
	if p.RefundedAt != nil {
		refundedTime = *p.RefundedAt
	}
	evt := events.NewEventFromCtx(ctx, events.PaymentRefundedTopic, "payments",
		events.PaymentRefundedEventPayload{
			OrderID:    p.OrderID,
			PaymentID:  p.ID,
			CustomerID: p.CustomerID,
			Amount:     p.Amount,
			RefundedAt: refundedTime,
		})
	s.publishEvent(ctx, evt)
}

// publishEvent stores the event in the outbox if available, otherwise publishes directly.
func (s *paymentService) publishEvent(ctx context.Context, evt events.Event) {
	if s.outbox != nil {
		if err := s.outbox.Store(ctx, evt); err != nil {
			logger.ErrorCtx(ctx, "Failed to store event %s in outbox: %v", evt.ID, err)
		}
		return
	}
	if s.eventBus != nil {
		s.eventBus.Publish(evt)
	}
}

func (s *paymentService) handleOrderCancelled(ev events.Event) error {
	var orderID string
	if payload, ok := events.AsPayload[events.OrderCancelledEventPayload](ev); ok {
		orderID = payload.OrderID
	} else if payloadVal, ok := ev.Payload.(events.OrderCancelledEventPayload); ok {
		orderID = payloadVal.OrderID
	} else if payloadPtr, ok := ev.Payload.(*events.OrderCancelledEventPayload); ok {
		orderID = payloadPtr.OrderID
	}

	if orderID == "" {
		logger.Error("handleOrderCancelled: received event payload with no order ID: %T", ev.Payload)
		return nil
	}

	logger.Info("handleOrderCancelled: received order.cancelled event for order %s. Initiating async refund.", orderID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := s.RefundPayment(ctx, orderID, nil)
	if err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			logger.Info("handleOrderCancelled: no payment found for order %s, no refund needed (order was probably unpaid)", orderID)
		} else {
			logger.Error("handleOrderCancelled: failed to refund payment for order %s: %v", orderID, err)
			return err
		}
	} else {
		logger.Info("handleOrderCancelled: successfully processed refund for order %s", orderID)
	}
	return nil
}
