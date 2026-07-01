// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package events implements the publish-subscribe messaging infrastructure,
// supporting local in-process delivery, retry semantics, dead-letter queues (DLQ),
// and integrations with Kafka and RabbitMQ.
package events

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/metrics"
)

var (
	// ErrBusClosed is returned when attempting to publish to a closed event bus.
	ErrBusClosed = errors.New("event bus is closed")
)

// LocalEventBus implements the EventBus interface for in-process, asynchronous
// event publishing and subscription with configurable retry logic and a Dead Letter Queue.
type LocalEventBus struct {
	mu             sync.RWMutex
	subscribers    map[string][]EventHandler
	closed         bool
	errorHandler   func(topic string, err error)
	dlq            chan Event
	wg             sync.WaitGroup
	maxRetries     int
	initialBackoff time.Duration
	maxBackoff     time.Duration
}

// NewEventBus constructs an EventBus based on the provided configuration.
// It switches implementations between Local, Kafka, and RabbitMQ backends.
func NewEventBus(cfg BusConfig) (EventBus, error) {
	switch cfg.Provider {
	case "kafka":
		return nil, errors.New("kafka event bus is not yet implemented")
	case "rabbitmq":
		return newRabbitMQEventBus(cfg.RabbitMQ, cfg.Retry)
	case "local", "":
		return NewLocalEventBusWithConfig(cfg.Local, cfg.Retry), nil
	default:
		return nil, fmt.Errorf("unknown event bus provider: %q", cfg.Provider)
	}
}

// NewLocalEventBus creates a new LocalEventBus using default retry and DLQ configurations.
func NewLocalEventBus() *LocalEventBus {
	return NewLocalEventBusWithConfig(LocalBusConfig{}, RetryConfig{})
}

// NewLocalEventBusWithConfig creates a new LocalEventBus using custom retry and DLQ settings.
func NewLocalEventBusWithConfig(cfg LocalBusConfig, retryCfg RetryConfig) *LocalEventBus {
	dlqSize := cfg.DLQBufferSize
	if dlqSize <= 0 {
		dlqSize = 100
	}
	maxRetries := retryCfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	initialBackoff := retryCfg.InitialBackoff
	if initialBackoff <= 0 {
		initialBackoff = 50 * time.Millisecond
	}
	maxBackoff := retryCfg.MaxBackoff
	if maxBackoff <= 0 {
		maxBackoff = 2 * time.Second
	}

	return &LocalEventBus{
		subscribers:    make(map[string][]EventHandler),
		dlq:            make(chan Event, dlqSize),
		maxRetries:     maxRetries,
		initialBackoff: initialBackoff,
		maxBackoff:     maxBackoff,
		errorHandler: func(topic string, err error) {
			logger.Error("Event handler failed for topic %s: %v", topic, err)
		},
	}
}

// WithErrorHandler overrides the default error handler for failed handler invocation.
func (b *LocalEventBus) WithErrorHandler(handler func(topic string, err error)) *LocalEventBus {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.errorHandler = handler
	return b
}

// DLQ returns the channel containing events that failed processing after all retry attempts.
func (b *LocalEventBus) DLQ() <-chan Event {
	return b.dlq
}

// Subscribe registers a handler to receive published events for a given topic.
func (b *LocalEventBus) Subscribe(topic string, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.subscribers[topic] = append(b.subscribers[topic], handler)
}

// Publish distributes an event asynchronously to all subscribed handlers on a separate goroutine.
func (b *LocalEventBus) Publish(event Event) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		logger.Warn("Event %s for topic %s dropped because event bus is closed", event.ID, event.Topic)
		return
	}
	handlers := append([]EventHandler(nil), b.subscribers[event.Topic]...)
	if len(handlers) == 0 {
		b.mu.RUnlock()
		logger.Warn("Event %s for topic %s dropped because no subscribers are registered", event.ID, event.Topic)
		return
	}
	b.wg.Add(len(handlers))
	b.mu.RUnlock()

	metrics.EventsPublishedTotal.WithLabelValues(event.Topic, event.Source).Inc()

	for _, handler := range handlers {
		go func(h EventHandler) {
			defer b.wg.Done()
			b.invokeHandlerWithRetry(h, event)
		}(handler)
	}
}

func (b *LocalEventBus) invokeHandlerWithRetry(handler EventHandler, event Event) {
	start := time.Now()
	dlqSent := false
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic recovered in retry loop: %v", r)
			b.handleError(event.Topic, err)
			if !dlqSent {
				b.sendToDLQ(event)
				metrics.EventsConsumedTotal.WithLabelValues(event.Topic, "failure").Inc()
				metrics.EventsHandlerDurationSeconds.WithLabelValues(event.Topic).Observe(time.Since(start).Seconds())
			}
		}
	}()

	// Retry mechanism
	var lastErr error
	maxRetries := b.maxRetries
	backoff := b.initialBackoff

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := b.executeHandler(handler, event)
		if err == nil {
			metrics.EventsConsumedTotal.WithLabelValues(event.Topic, "success").Inc()
			metrics.EventsHandlerDurationSeconds.WithLabelValues(event.Topic).Observe(time.Since(start).Seconds())
			return
		}
		lastErr = err
		logger.Warn("Event handler failed for topic %s (attempt %d/%d): %v", event.Topic, attempt, maxRetries, err)
		metrics.EventsRetriesTotal.WithLabelValues(event.Topic, "local", strconv.Itoa(attempt)).Inc()
		if attempt < maxRetries {
			time.Sleep(backoff)
			backoff *= 2
			if backoff > b.maxBackoff {
				backoff = b.maxBackoff
			}
		}
	}

	logger.Error("Event handler failed after %d retries for topic %s: %v. Sending to DLQ.", maxRetries, event.Topic, lastErr)
	b.handleError(event.Topic, lastErr)
	dlqSent = true
	b.sendToDLQ(event)
	metrics.EventsConsumedTotal.WithLabelValues(event.Topic, "failure").Inc()
	metrics.EventsHandlerDurationSeconds.WithLabelValues(event.Topic).Observe(time.Since(start).Seconds())
}

func (b *LocalEventBus) handleError(topic string, err error) {
	b.mu.RLock()
	errHandler := b.errorHandler
	b.mu.RUnlock()
	if errHandler != nil {
		errHandler(topic, err)
	}
}

// executeHandler captures panics from executing the actual handler function
func (b *LocalEventBus) executeHandler(handler EventHandler, event Event) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return handler(event)
}

func (b *LocalEventBus) sendToDLQ(event Event) {
	select {
	case b.dlq <- event:
		logger.Warn("Event %s sent to DLQ (Topic: %s)", event.ID, event.Topic)
		metrics.EventsDLQTotal.WithLabelValues(event.Topic, "local").Inc()
	default:
		logger.Error("DLQ is full, dropped event %s (Topic: %s)", event.ID, event.Topic)
		metrics.EventsDLQDroppedTotal.WithLabelValues(event.Topic).Inc()
	}
}

// PublishCtx publishes events while verifying that the bus is not closed, accepting a context for future extensions.
func (b *LocalEventBus) PublishCtx(ctx context.Context, event Event) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return ErrBusClosed
	}
	b.mu.RUnlock()
	b.Publish(event)
	return nil
}

// Close transitions the bus state to closed and closes the dead letter queue channel.
func (b *LocalEventBus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.mu.Unlock()
	b.wg.Wait()
	close(b.dlq)
	return nil
}

var _ EventBus = (*LocalEventBus)(nil)
