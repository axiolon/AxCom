// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package orders contains the core service, domain models, validation, and state machine transitions for managing orders in the system.
package orders

import (
	"context"
	"ecom-engine/internal/core/orders/domain"
	"errors"
	"sync"
	"testing"
	"time"

	"ecom-engine/internal/events"
	apperrors "ecom-engine/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockOrderRepository implements OrderRepository with thread safety
type MockOrderRepository struct {
	mu          sync.RWMutex
	orders      map[string]*Order
	createErr   error
	getErr      error
	updateErr   error
	listCustErr error
	listAllErr  error
}

func NewMockOrderRepository() *MockOrderRepository {
	return &MockOrderRepository{
		orders: make(map[string]*Order),
	}
}

func (m *MockOrderRepository) Create(_ context.Context, o *Order) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr != nil {
		return m.createErr
	}
	m.orders[o.ID] = o
	return nil
}

func (m *MockOrderRepository) GetByID(_ context.Context, id string) (*Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getErr != nil {
		return nil, m.getErr
	}
	o, exists := m.orders[id]
	if !exists {
		return nil, domain.ErrOrderNotFound
	}
	return o, nil
}

func (m *MockOrderRepository) Update(_ context.Context, o *Order) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateErr != nil {
		return m.updateErr
	}
	m.orders[o.ID] = o
	return nil
}

func (m *MockOrderRepository) ListByCustomerID(_ context.Context, customerID string, limit, offset int) ([]Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.listCustErr != nil {
		return nil, m.listCustErr
	}
	var list []Order
	for _, o := range m.orders {
		if o.CustomerID == customerID {
			list = append(list, *o)
		}
	}

	if offset > len(list) {
		return []Order{}, nil
	}
	end := offset + limit
	if end > len(list) {
		end = len(list)
	}
	return list[offset:end], nil
}

func (m *MockOrderRepository) ListAll(_ context.Context, limit, offset int) ([]Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.listAllErr != nil {
		return nil, m.listAllErr
	}
	var list []Order
	for _, o := range m.orders {
		list = append(list, *o)
	}

	if offset > len(list) {
		return []Order{}, nil
	}
	end := offset + limit
	if end > len(list) {
		end = len(list)
	}
	return list[offset:end], nil
}

func (m *MockOrderRepository) CountByStatus(_ context.Context) (map[string]int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	counts := make(map[string]int64)
	for _, o := range m.orders {
		counts[string(o.Status)]++
	}
	return counts, nil
}

func (m *MockOrderRepository) SumRevenue(_ context.Context, since time.Time) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var total float64
	for _, o := range m.orders {
		if since.IsZero() || o.CreatedAt.After(since) || o.CreatedAt.Equal(since) {
			total += o.Total
		}
	}
	return total, nil
}

func (m *MockOrderRepository) RevenueByDay(_ context.Context, _ int) ([]DailyRevenue, error) {
	return nil, nil
}

func (m *MockOrderRepository) TopProducts(_ context.Context, _ int) ([]ProductSales, error) {
	return nil, nil
}

// MockEventBus implements events.EventBus with synchronous triggers for fast testing
type MockEventBus struct {
	mu          sync.RWMutex
	published   []events.Event
	subscribers map[string][]events.EventHandler
}

func NewMockEventBus() *MockEventBus {
	return &MockEventBus{
		subscribers: make(map[string][]events.EventHandler),
	}
}

func (m *MockEventBus) Subscribe(topic string, handler events.EventHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribers[topic] = append(m.subscribers[topic], handler)
}

func (m *MockEventBus) Publish(event events.Event) {
	m.mu.Lock()
	m.published = append(m.published, event)
	m.mu.Unlock()

	m.mu.RLock()
	handlers, exists := m.subscribers[event.Topic]
	m.mu.RUnlock()
	if exists {
		for _, handler := range handlers {
			_ = handler(event) // Execute synchronously in test
		}
	}
}

func (m *MockEventBus) Close() error {
	return nil
}

func (m *MockEventBus) GetPublished() []events.Event {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.published
}

func TestOrderService_CreateOrder(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*MockOrderRepository, *MockEventBus, Service) {
		repo := NewMockOrderRepository()
		bus := NewMockEventBus()
		svc := NewOrderService(repo, bus, nil, nil)
		return repo, bus, svc
	}

	t.Run("successful order creation", func(t *testing.T) {
		t.Parallel()
		repo, bus, svc := setup(t)

		items := []OrderItem{
			{VariantID: "var_1", Quantity: 2, Price: 15.50},
			{VariantID: "var_2", Quantity: 1, Price: 9.00},
		}

		o, err := svc.CreateOrder(context.Background(), "cust_123", OrderCustomerSnapshot{}, items)
		require.NoError(t, err)
		assert.NotEmpty(t, o.ID)
		assert.Equal(t, "cust_123", o.CustomerID)
		assert.Equal(t, 40.00, o.Total) // (15.5 * 2) + 9 = 40.00
		assert.Equal(t, StatusPending, o.Status)
		assert.Len(t, o.Items, 2)

		// Verify repo persistence
		stored, err := repo.GetByID(context.Background(), o.ID)
		require.NoError(t, err)
		assert.Equal(t, o.ID, stored.ID)

		// Verify published event
		publishedEvents := bus.GetPublished()
		require.Len(t, publishedEvents, 1)
		assert.Equal(t, events.OrderCreatedTopic, publishedEvents[0].Topic)
	})

	t.Run("validation error - empty items list", func(t *testing.T) {
		t.Parallel()
		_, _, svc := setup(t)

		_, err := svc.CreateOrder(context.Background(), "cust_123", OrderCustomerSnapshot{}, []OrderItem{})
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.Code)
		assert.True(t, errors.Is(appErr.Err, domain.ErrEmptyOrder))
	})

	t.Run("validation error - missing variant id", func(t *testing.T) {
		t.Parallel()
		_, _, svc := setup(t)

		items := []OrderItem{{VariantID: "", Quantity: 1, Price: 10.0}}
		_, err := svc.CreateOrder(context.Background(), "cust_123", OrderCustomerSnapshot{}, items)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.Code)
		assert.True(t, errors.Is(appErr.Err, domain.ErrVariantIDRequired))
	})

	t.Run("validation error - non-positive quantity", func(t *testing.T) {
		t.Parallel()
		_, _, svc := setup(t)

		items := []OrderItem{{VariantID: "v1", Quantity: 0, Price: 10.0}}
		_, err := svc.CreateOrder(context.Background(), "cust_123", OrderCustomerSnapshot{}, items)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.Code)
		assert.True(t, errors.Is(appErr.Err, domain.ErrInvalidQuantity))
	})

	t.Run("validation error - negative price", func(t *testing.T) {
		t.Parallel()
		_, _, svc := setup(t)

		items := []OrderItem{{VariantID: "v1", Quantity: 1, Price: -5.0}}
		_, err := svc.CreateOrder(context.Background(), "cust_123", OrderCustomerSnapshot{}, items)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.Code)
		assert.True(t, errors.Is(appErr.Err, domain.ErrInvalidPrice))
	})

	t.Run("repository error on create", func(t *testing.T) {
		t.Parallel()
		repo, _, svc := setup(t)
		repo.createErr = errors.New("database unavailable")

		items := []OrderItem{{VariantID: "var_1", Quantity: 1, Price: 10.00}}
		_, err := svc.CreateOrder(context.Background(), "cust_123", OrderCustomerSnapshot{}, items)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 500, appErr.Code)
	})
}

func TestOrderService_TransitionOrder(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*MockOrderRepository, *MockEventBus, Service) {
		repo := NewMockOrderRepository()
		bus := NewMockEventBus()
		svc := NewOrderService(repo, bus, nil, nil)
		return repo, bus, svc
	}

	t.Run("successful transition to paid", func(t *testing.T) {
		t.Parallel()
		repo, bus, svc := setup(t)

		order := &Order{
			ID:         "ord_123",
			CustomerID: "cust_1",
			Status:     StatusPending,
			Total:      100.00,
		}
		repo.orders[order.ID] = order

		updated, err := svc.TransitionOrder(context.Background(), "ord_123", "pay")
		require.NoError(t, err)
		assert.Equal(t, StatusPaid, updated.Status)

		// Verify event was published
		publishedEvents := bus.GetPublished()
		require.Len(t, publishedEvents, 1)
		assert.Equal(t, events.OrderPaidTopic, publishedEvents[0].Topic)
	})

	t.Run("transition fails - order not found", func(t *testing.T) {
		t.Parallel()
		_, _, svc := setup(t)

		_, err := svc.TransitionOrder(context.Background(), "ord_missing", "pay")
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.Code)
		assert.True(t, errors.Is(appErr.Err, domain.ErrOrderNotFound))
	})

	t.Run("transition fails - invalid state transition action", func(t *testing.T) {
		t.Parallel()
		repo, _, svc := setup(t)

		order := &Order{
			ID:     "ord_123",
			Status: StatusPending,
		}
		repo.orders[order.ID] = order

		// Attempt to complete order directly from pending
		_, err := svc.TransitionOrder(context.Background(), "ord_123", "complete")
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.Code)
	})

	t.Run("transition fails - update repository error", func(t *testing.T) {
		t.Parallel()
		repo, _, svc := setup(t)

		order := &Order{
			ID:     "ord_123",
			Status: StatusPending,
		}
		repo.orders[order.ID] = order
		repo.updateErr = errors.New("failed to write update")

		_, err := svc.TransitionOrder(context.Background(), "ord_123", "pay")
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 500, appErr.Code)
	})
}

func TestOrderService_GetOrder(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*MockOrderRepository, Service) {
		repo := NewMockOrderRepository()
		bus := NewMockEventBus()
		svc := NewOrderService(repo, bus, nil, nil)
		return repo, svc
	}

	t.Run("retrieve order success", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		order := &Order{ID: "ord_1", CustomerID: "cust_1"}
		repo.orders[order.ID] = order

		o, err := svc.GetOrder(context.Background(), "ord_1")
		require.NoError(t, err)
		assert.Equal(t, "ord_1", o.ID)
	})

	t.Run("retrieve order not found", func(t *testing.T) {
		t.Parallel()
		_, svc := setup(t)

		_, err := svc.GetOrder(context.Background(), "ord_missing")
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.Code)
	})
}

func TestOrderService_GetCustomerOrders(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*MockOrderRepository, Service) {
		repo := NewMockOrderRepository()
		bus := NewMockEventBus()
		svc := NewOrderService(repo, bus, nil, nil)
		return repo, svc
	}

	t.Run("retrieve customer orders success with pagination", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.orders["ord_1"] = &Order{ID: "ord_1", CustomerID: "c1"}
		repo.orders["ord_2"] = &Order{ID: "ord_2", CustomerID: "c1"}
		repo.orders["ord_3"] = &Order{ID: "ord_3", CustomerID: "c1"}
		repo.orders["ord_4"] = &Order{ID: "ord_4", CustomerID: "c2"}

		// Page 1: limit 2, offset 0 -> returns 2
		list, err := svc.GetCustomerOrders(context.Background(), "c1", 2, 0)
		require.NoError(t, err)
		assert.Len(t, list, 2)

		// Page 2: limit 2, offset 2 -> returns 1
		list2, err := svc.GetCustomerOrders(context.Background(), "c1", 2, 2)
		require.NoError(t, err)
		assert.Len(t, list2, 1)
	})

	t.Run("retrieve customer orders failure", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)
		repo.listCustErr = errors.New("list failed")

		_, err := svc.GetCustomerOrders(context.Background(), "c1", 10, 0)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 500, appErr.Code)
	})
}

func TestOrderService_GetAllOrders(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*MockOrderRepository, Service) {
		repo := NewMockOrderRepository()
		bus := NewMockEventBus()
		svc := NewOrderService(repo, bus, nil, nil)
		return repo, svc
	}

	t.Run("get all orders success with pagination", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.orders["ord_1"] = &Order{ID: "ord_1"}
		repo.orders["ord_2"] = &Order{ID: "ord_2"}
		repo.orders["ord_3"] = &Order{ID: "ord_3"}

		list, err := svc.GetAllOrders(context.Background(), 2, 0)
		require.NoError(t, err)
		assert.Len(t, list, 2)

		list2, err := svc.GetAllOrders(context.Background(), 2, 2)
		require.NoError(t, err)
		assert.Len(t, list2, 1)
	})

	t.Run("get all orders failure", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)
		repo.listAllErr = errors.New("failed to list all")

		_, err := svc.GetAllOrders(context.Background(), 10, 0)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 500, appErr.Code)
	})
}

func TestOrderService_HandlePaymentSucceeded(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*MockOrderRepository, *MockEventBus, Service) {
		repo := NewMockOrderRepository()
		bus := NewMockEventBus()
		svc := NewOrderService(repo, bus, nil, nil)
		return repo, bus, svc
	}

	t.Run("payment succeeded event triggers state transition to paid", func(t *testing.T) {
		t.Parallel()
		repo, bus, _ := setup(t)

		order := &Order{
			ID:     "ord_pay_event",
			Status: StatusPending,
		}
		repo.orders[order.ID] = order

		// Publish event to mock bus, which triggers synchronous handler callback
		bus.Publish(events.Event{
			Topic:     events.PaymentSucceededTopic,
			Timestamp: time.Now(),
			Payload: events.PaymentEventPayload{
				OrderID: "ord_pay_event",
				Amount:  100.00,
			},
		})

		// Give a tiny amount of time for the goroutine in event handler context (if any) or just read directly.
		// Wait, the handler inside `handlePaymentSucceeded` spans a goroutine: "go s.TransitionOrder(...)"
		// No, `handlePaymentSucceeded` spans a goroutine: `go s.TransitionOrder(ctx, payload.OrderID, "pay")`?
		// Wait, let's look at service.go line 177:
		// `ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)`
		// `defer cancel()`
		// `_, err := s.TransitionOrder(ctx, payload.OrderID, "pay")`
		// Wait, it DOES NOT spin a goroutine inside handlePaymentSucceeded! The LocalEventBus spans a goroutine when calling handler:
		// `go handler(event)`
		// Since we call `handler(event)` directly synchronously in our `MockEventBus`, it executes blocking in the current test goroutine.
		// So the transition should be completed immediately!
		stored, err := repo.GetByID(context.Background(), "ord_pay_event")
		require.NoError(t, err)
		assert.Equal(t, StatusPaid, stored.Status)
	})
}
