// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"

	"ecom-engine/pkg/ctxkeys"
	"ecom-engine/pkg/idgen"
)

type Event struct {
	ID            string    `json:"id"`
	Topic         string    `json:"topic"`
	Source        string    `json:"source"`
	Version       int       `json:"version"`
	Timestamp     time.Time `json:"timestamp"`
	Payload       any       `json:"payload"`
	TraceID       string    `json:"trace_id,omitempty"`
	CorrelationID string    `json:"correlation_id,omitempty"`
}

const (
	OrderCreatedTopic          = "order.created"
	OrderPaidTopic             = "order.paid"
	InventoryLowTopic          = "inventory.low"
	UserRegisteredTopic        = "user.registered"
	InventoryStockChangedTopic = "inventory.stock_changed"
	PaymentSucceededTopic      = "payment.succeeded"
	PaymentFailedTopic         = "payment.failed"
	PaymentRefundedTopic       = "payment.refunded"
	OrderShippedTopic          = "order.shipped"
	OrderCancelledTopic        = "order.cancelled"
)

// Typed Event Payloads

type OrderCreatedEventPayload struct {
	OrderID    string    `json:"order_id"`
	CustomerID string    `json:"customer_id"`
	Total      float64   `json:"total"`
	CreatedAt  time.Time `json:"created_at"`
}

type OrderPaidEventPayload struct {
	OrderID string  `json:"order_id"`
	Amount  float64 `json:"amount"`
}

type OrderCancelledEventPayload struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason,omitempty"`
}

type OrderShippedEventPayload struct {
	OrderID        string `json:"order_id"`
	TrackingNumber string `json:"tracking_number,omitempty"`
	Carrier        string `json:"carrier,omitempty"`
}

type StockChangedPayload struct {
	VariantID    string `json:"variant_id"`
	LocationID   string `json:"location_id"`
	OldQuantity  int    `json:"old_quantity"`
	NewQuantity  int    `json:"new_quantity"`
	ChangeReason string `json:"change_reason"`
	ChangedBy    string `json:"changed_by"`
}

type PaymentEventPayload struct {
	OrderID    string  `json:"order_id"`
	PaymentID  string  `json:"payment_id"`
	CustomerID string  `json:"customer_id"`
	Amount     float64 `json:"amount"`
}

type PaymentFailedEventPayload struct {
	OrderID    string  `json:"order_id"`
	PaymentID  string  `json:"payment_id"`
	CustomerID string  `json:"customer_id"`
	Amount     float64 `json:"amount"`
	Error      string  `json:"error"`
}

type PaymentRefundedEventPayload struct {
	OrderID    string    `json:"order_id"`
	PaymentID  string    `json:"payment_id"`
	CustomerID string    `json:"customer_id"`
	Amount     float64   `json:"amount"`
	RefundedAt time.Time `json:"refunded_at"`
}

type UserRegisteredPayload struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type InventoryLowPayload struct {
	VariantID  string `json:"variant_id"`
	LocationID string `json:"location_id"`
	Quantity   int    `json:"quantity"`
}

// NewEvent builds a standard Event envelope with default metadata.
func NewEvent(topic, source string, payload any) Event {
	return Event{
		ID:        idgen.MustGenerate("evt_"),
		Topic:     topic,
		Source:    source,
		Version:   1,
		Timestamp: time.Now(),
		Payload:   payload,
	}
}

// NewEventFromCtx builds a standard Event envelope and populates TraceID from context spans.
func NewEventFromCtx(ctx context.Context, topic, source string, payload any) Event {
	evt := NewEvent(topic, source, payload)
	if ctx != nil {
		spanCtx := trace.SpanContextFromContext(ctx)
		if spanCtx.IsValid() {
			evt.TraceID = spanCtx.TraceID().String()
		}
		// Try to fetch context-based correlation ID if stored
		if corrID, ok := ctx.Value(ctxkeys.CorrelationIDKey).(string); ok {
			evt.CorrelationID = corrID
		}
	}
	return evt
}

// AsPayload attempts to convert the Event's payload into type T.
func AsPayload[T any](event Event) (T, bool) {
	val, ok := event.Payload.(T)
	return val, ok
}

type EventHandler func(event Event) error

type EventBus interface {
	Subscribe(topic string, handler EventHandler)
	Publish(event Event)
	Close() error
}
