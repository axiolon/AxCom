// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package orders contains the core service, domain models, validation, and state machine transitions for managing orders in the system.
package orders

import (
	"context"
	"ecom-engine/internal/core/orders/domain"
	"ecom-engine/internal/events"
	infradb "ecom-engine/internal/infra/db"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/idgen"
	"ecom-engine/pkg/logger"
	"time"
)

// Service defines the business logic contract for managing orders.
type Service interface {
	// CreateOrder validates items, calculates totals, persists the order, and publishes events.
	CreateOrder(ctx context.Context, customerID string, customerSnapshot OrderCustomerSnapshot, items []OrderItem) (*Order, error)

	// TransitionOrder attempts to transition the status of an order using actions.
	TransitionOrder(ctx context.Context, id string, action string) (*Order, error)

	// GetOrder retrieves a specific order by its ID.
	GetOrder(ctx context.Context, id string) (*Order, error)

	// GetCustomerOrders lists all orders for a specific customer ID.
	GetCustomerOrders(ctx context.Context, customerID string, limit, offset int) ([]Order, error)

	// GetAllOrders retrieves all orders globally.
	GetAllOrders(ctx context.Context, limit, offset int) ([]Order, error)
}

type orderService struct {
	repo         OrderRepository
	stateMachine *OrderStateMachine
	eventBus     events.EventBus
	outbox       events.OutboxRepository
	txManager    infradb.TransactionManager
}

// NewOrderService creates a new instance of the order Service.
// outbox and txManager may be nil when the outbox feature is disabled.
func NewOrderService(repo OrderRepository, eventBus events.EventBus, outbox events.OutboxRepository, txManager infradb.TransactionManager) Service {
	s := &orderService{
		repo:         repo,
		stateMachine: NewOrderStateMachine(),
		eventBus:     eventBus,
		outbox:       outbox,
		txManager:    txManager,
	}
	eventBus.Subscribe(events.PaymentSucceededTopic, s.handlePaymentSucceeded)
	return s
}

// CreateOrder validates items, calculates totals, persists the order, and publishes events.
func (s *orderService) CreateOrder(ctx context.Context, customerID string, customerSnapshot OrderCustomerSnapshot, items []OrderItem) (*Order, error) {
	logger.InfoCtx(ctx, "Creating order for customer %s with %d items", customerID, len(items))

	if err := domain.ValidateItems(items); err != nil {
		logger.ErrorCtx(ctx, "Order validation failed for customer %s: %v", customerID, err)
		switch err {
		case domain.ErrEmptyOrder:
			return nil, apperrors.NewBadRequest("order must contain at least one item", domain.ErrEmptyOrder)
		case domain.ErrVariantIDRequired:
			return nil, apperrors.NewBadRequest("variant ID is required for each item", domain.ErrVariantIDRequired)
		case domain.ErrInvalidQuantity:
			return nil, apperrors.NewBadRequest("quantity must be greater than zero", domain.ErrInvalidQuantity)
		case domain.ErrInvalidPrice:
			return nil, apperrors.NewBadRequest("price cannot be negative", domain.ErrInvalidPrice)
		default:
			return nil, apperrors.NewBadRequest("invalid order items", err)
		}
	}

	total := domain.CalculateTotal(items)

	ordID, err := idgen.Generate("ord_")
	if err != nil {
		return nil, apperrors.NewInternal("failed to generate order ID", err)
	}

	order := &Order{
		ID:               ordID,
		CustomerID:       customerID,
		CustomerSnapshot: customerSnapshot,
		Items:            items,
		Total:            total,
		Status:           StatusPending,
		CreatedAt:        time.Now(),
	}

	evt := events.NewEventFromCtx(ctx, events.OrderCreatedTopic, "orders",
		events.OrderCreatedEventPayload{
			OrderID:    order.ID,
			CustomerID: order.CustomerID,
			Total:      order.Total,
			CreatedAt:  order.CreatedAt,
		})

	if s.outbox != nil && s.txManager != nil {
		if err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
			if err := s.repo.Create(txCtx, order); err != nil {
				return err
			}
			return s.outbox.Store(txCtx, evt)
		}); err != nil {
			logger.ErrorCtx(ctx, "Failed to create order in repo: %v", err)
			return nil, apperrors.NewInternal("failed to create order", err)
		}
	} else {
		if err := s.repo.Create(ctx, order); err != nil {
			logger.ErrorCtx(ctx, "Failed to create order in repo: %v", err)
			return nil, apperrors.NewInternal("failed to create order", err)
		}
		s.eventBus.Publish(evt)
	}

	logger.InfoCtx(ctx, "Successfully created order %s", order.ID)
	return order, nil
}

// TransitionOrder attempts to transition the status of an order using actions.
func (s *orderService) TransitionOrder(ctx context.Context, id string, action string) (*Order, error) {
	logger.InfoCtx(ctx, "Transitioning order %s with action %s", id, action)

	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to retrieve order %s for transition: %v", id, err)
		return nil, apperrors.NewNotFound("order not found", domain.ErrOrderNotFound)
	}

	nextStatus, err := s.stateMachine.Transition(order.Status, action)
	if err != nil {
		logger.ErrorCtx(ctx, "State machine transition failed for order %s: %v", id, err)
		return nil, apperrors.NewBadRequest("invalid state transition action", domain.ErrInvalidTransition)
	}

	order.Status = nextStatus

	// Build event for the transition (if any).
	var evt *events.Event
	switch nextStatus {
	case StatusPaid:
		e := events.NewEventFromCtx(ctx, events.OrderPaidTopic, "orders",
			events.OrderPaidEventPayload{OrderID: order.ID, Amount: order.Total})
		evt = &e
	case StatusCanceled:
		e := events.NewEventFromCtx(ctx, events.OrderCancelledTopic, "orders",
			events.OrderCancelledEventPayload{OrderID: order.ID, Reason: "order cancelled"})
		evt = &e
	}

	if evt != nil && s.outbox != nil && s.txManager != nil {
		if err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
			if err := s.repo.Update(txCtx, order); err != nil {
				return err
			}
			return s.outbox.Store(txCtx, *evt)
		}); err != nil {
			logger.ErrorCtx(ctx, "Failed to update order %s status to %s: %v", id, nextStatus, err)
			return nil, apperrors.NewInternal("failed to update order status", err)
		}
	} else {
		if err := s.repo.Update(ctx, order); err != nil {
			logger.ErrorCtx(ctx, "Failed to update order %s status to %s: %v", id, nextStatus, err)
			return nil, apperrors.NewInternal("failed to update order status", err)
		}
		if evt != nil {
			s.eventBus.Publish(*evt)
		}
	}

	logger.InfoCtx(ctx, "Successfully transitioned order %s to %s", id, nextStatus)
	return order, nil
}

// GetOrder retrieves a specific order by its ID.
func (s *orderService) GetOrder(ctx context.Context, id string) (*Order, error) {
	logger.InfoCtx(ctx, "Retrieving order with ID: %s", id)

	o, err := s.repo.GetByID(ctx, id)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to retrieve order %s: %v", id, err)
		return nil, apperrors.NewNotFound("order not found", domain.ErrOrderNotFound)
	}

	return o, nil
}

// GetCustomerOrders lists all orders for a specific customer ID.
func (s *orderService) GetCustomerOrders(ctx context.Context, customerID string, limit, offset int) ([]Order, error) {
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	logger.InfoCtx(ctx, "Listing orders for customer: %s, limit: %d, offset: %d", customerID, limit, offset)

	orders, err := s.repo.ListByCustomerID(ctx, customerID, limit, offset)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to list orders for customer %s: %v", customerID, err)
		return nil, apperrors.NewInternal("failed to list customer orders", err)
	}

	return orders, nil
}

// GetAllOrders retrieves all orders globally.
func (s *orderService) GetAllOrders(ctx context.Context, limit, offset int) ([]Order, error) {
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	logger.InfoCtx(ctx, "Listing all orders globally, limit: %d, offset: %d", limit, offset)

	ordersList, err := s.repo.ListAll(ctx, limit, offset)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to list all orders: %v", err)
		return nil, apperrors.NewInternal("failed to list all orders", err)
	}

	return ordersList, nil
}

func (s *orderService) handlePaymentSucceeded(ev events.Event) error {
	payload, ok := events.AsPayload[events.PaymentEventPayload](ev)
	if !ok {
		logger.Error("Invalid payload type for PaymentSucceededTopic: expected PaymentEventPayload, got %T", ev.Payload)
		return nil
	}

	logger.Info("Order payment success handler triggered for order: %s, amount: %.2f", payload.OrderID, payload.Amount)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := s.TransitionOrder(ctx, payload.OrderID, "pay")
	if err != nil {
		logger.Error("Failed to transition order %s to paid: %v", payload.OrderID, err)
		return err
	}
	return nil
}
