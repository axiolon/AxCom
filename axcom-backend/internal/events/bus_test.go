// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ecom-engine/pkg/ctxkeys"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestLocalEventBus_PublishSubscribe(t *testing.T) {
	bus := NewLocalEventBus()
	defer func() { _ = bus.Close() }()

	var received Event
	done := make(chan struct{})

	bus.Subscribe("test.topic", func(ev Event) error {
		received = ev
		close(done)
		return nil
	})

	evt := NewEvent("test.topic", "test_source", "hello")
	bus.Publish(evt)

	select {
	case <-done:
		assert.Equal(t, "test.topic", received.Topic)
		assert.Equal(t, "test_source", received.Source)
		assert.Equal(t, "hello", received.Payload)
		assert.NotEmpty(t, received.ID)
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestLocalEventBus_AsPayloadHelper(t *testing.T) {
	evt := NewEvent("test.topic", "test_source", PaymentEventPayload{
		OrderID:    "ord_1",
		PaymentID:  "pmt_1",
		CustomerID: "cust_1",
		Amount:     10.5,
	})

	payload, ok := AsPayload[PaymentEventPayload](evt)
	require.True(t, ok)
	assert.Equal(t, "ord_1", payload.OrderID)
	assert.Equal(t, 10.5, payload.Amount)

	_, ok = AsPayload[StockChangedPayload](evt)
	assert.False(t, ok)
}

func TestLocalEventBus_RetryAndDLQ_Panic(t *testing.T) {
	bus := NewLocalEventBus()
	defer func() { _ = bus.Close() }()

	var callCount int32
	bus.Subscribe("retry.topic", func(_ Event) error {
		atomic.AddInt32(&callCount, 1)
		panic("simulated failure")
	})

	evt := NewEvent("retry.topic", "test_source", "data")
	bus.Publish(evt)

	select {
	case dlqEvt := <-bus.DLQ():
		assert.Equal(t, "retry.topic", dlqEvt.Topic)
		assert.Equal(t, "data", dlqEvt.Payload)
		assert.EqualValues(t, 3, atomic.LoadInt32(&callCount))
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event in DLQ")
	}
}

func TestLocalEventBus_RetryAndDLQ_ReturnedError(t *testing.T) {
	bus := NewLocalEventBusWithConfig(LocalBusConfig{DLQBufferSize: 2}, RetryConfig{
		MaxRetries:     2,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     time.Millisecond,
	})
	defer func() { _ = bus.Close() }()

	var callCount int32
	bus.Subscribe("error.topic", func(_ Event) error {
		atomic.AddInt32(&callCount, 1)
		return errors.New("transient failure")
	})

	bus.Publish(NewEvent("error.topic", "test_source", "data"))

	select {
	case dlqEvt := <-bus.DLQ():
		assert.Equal(t, "error.topic", dlqEvt.Topic)
		assert.EqualValues(t, 2, atomic.LoadInt32(&callCount))
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event in DLQ")
	}
}

func TestLocalEventBus_ClosedChecks(t *testing.T) {
	bus := NewLocalEventBus()
	err := bus.Close()
	require.NoError(t, err)

	evt := NewEvent("test.topic", "test_source", "hello")
	err = bus.PublishCtx(context.Background(), evt)
	assert.ErrorIs(t, err, ErrBusClosed)
}

func TestLocalEventBus_PublishCtxCancelled(t *testing.T) {
	bus := NewLocalEventBus()
	defer func() { _ = bus.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := bus.PublishCtx(ctx, NewEvent("test.topic", "test_source", "hello"))
	assert.ErrorIs(t, err, context.Canceled)
}

func TestLocalEventBus_CustomConfig(t *testing.T) {
	retryCfg := RetryConfig{
		MaxRetries:     2,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     50 * time.Millisecond,
	}
	localCfg := LocalBusConfig{
		DLQBufferSize: 5,
	}

	bus := NewLocalEventBusWithConfig(localCfg, retryCfg)
	defer func() { _ = bus.Close() }()

	assert.Equal(t, 2, bus.maxRetries)
	assert.Equal(t, 10*time.Millisecond, bus.initialBackoff)
	assert.Equal(t, 50*time.Millisecond, bus.maxBackoff)
	assert.Equal(t, 5, cap(bus.dlq))

	var callCount int32
	bus.Subscribe("custom.topic", func(_ Event) error {
		atomic.AddInt32(&callCount, 1)
		panic("fail")
	})

	evt := NewEvent("custom.topic", "test", "data")
	bus.Publish(evt)

	select {
	case dlqEvt := <-bus.DLQ():
		assert.Equal(t, "custom.topic", dlqEvt.Topic)
		assert.EqualValues(t, 2, atomic.LoadInt32(&callCount))
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for event in DLQ")
	}
}

func TestLocalEventBus_MultipleSubscribersSameTopic(t *testing.T) {
	bus := NewLocalEventBus()
	defer func() { _ = bus.Close() }()

	var calls int32
	done := make(chan struct{})

	for i := 0; i < 2; i++ {
		bus.Subscribe("shared.topic", func(_ Event) error {
			if atomic.AddInt32(&calls, 1) == 2 {
				close(done)
			}
			return nil
		})
	}

	bus.Publish(NewEvent("shared.topic", "test", nil))

	select {
	case <-done:
		assert.EqualValues(t, 2, atomic.LoadInt32(&calls))
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for subscribers")
	}
}

func TestLocalEventBus_MultipleTopics(t *testing.T) {
	bus := NewLocalEventBus()
	defer func() { _ = bus.Close() }()

	var topicA, topicB int32
	doneA := make(chan struct{})
	doneB := make(chan struct{})

	bus.Subscribe("topic.a", func(_ Event) error {
		atomic.AddInt32(&topicA, 1)
		close(doneA)
		return nil
	})
	bus.Subscribe("topic.b", func(_ Event) error {
		atomic.AddInt32(&topicB, 1)
		close(doneB)
		return nil
	})

	bus.Publish(NewEvent("topic.a", "test", nil))
	bus.Publish(NewEvent("topic.b", "test", nil))

	select {
	case <-doneA:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for topic.a")
	}

	select {
	case <-doneB:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for topic.b")
	}

	assert.EqualValues(t, 1, atomic.LoadInt32(&topicA))
	assert.EqualValues(t, 1, atomic.LoadInt32(&topicB))
}

func TestLocalEventBus_ConcurrentPublishStress(t *testing.T) {
	bus := NewLocalEventBus()
	defer func() { _ = bus.Close() }()

	const eventCount = 100
	var calls int32
	done := make(chan struct{})

	bus.Subscribe("stress.topic", func(_ Event) error {
		if atomic.AddInt32(&calls, 1) == eventCount {
			close(done)
		}
		return nil
	})

	var wg sync.WaitGroup
	for i := 0; i < eventCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(NewEvent("stress.topic", "test", nil))
		}()
	}
	wg.Wait()

	select {
	case <-done:
		assert.EqualValues(t, eventCount, atomic.LoadInt32(&calls))
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for concurrent publishes")
	}
}

func TestLocalEventBus_DLQFullDropsWithoutBlocking(t *testing.T) {
	bus := NewLocalEventBusWithConfig(LocalBusConfig{DLQBufferSize: 1}, RetryConfig{
		MaxRetries:     1,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     time.Millisecond,
	})
	defer func() { _ = bus.Close() }()

	var calls int32
	done := make(chan struct{})
	bus.Subscribe("dlq.full", func(_ Event) error {
		if atomic.AddInt32(&calls, 1) == 2 {
			close(done)
		}
		return errors.New("fail")
	})

	bus.Publish(NewEvent("dlq.full", "test", "one"))

	require.Eventually(t, func() bool {
		return len(bus.dlq) == 1
	}, time.Second, time.Millisecond)

	bus.Publish(NewEvent("dlq.full", "test", "two"))

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for failed handlers")
	}

	// Give the second event's goroutine time to complete its sendToDLQ attempt
	time.Sleep(50 * time.Millisecond)

	select {
	case extra := <-bus.DLQ():
		assert.NotEmpty(t, extra.ID)
	case <-time.After(25 * time.Millisecond):
		t.Fatal("first DLQ event missing")
	}

	select {
	case extra := <-bus.DLQ():
		t.Fatalf("expected second DLQ event to be dropped, got %s", extra.ID)
	case <-time.After(25 * time.Millisecond):
	}
}

func TestLocalEventBus_CloseDrainsInFlightHandlers(t *testing.T) {
	bus := NewLocalEventBus()

	started := make(chan struct{})
	release := make(chan struct{})
	closed := make(chan struct{})

	bus.Subscribe("slow.topic", func(_ Event) error {
		close(started)
		<-release
		return nil
	})

	bus.Publish(NewEvent("slow.topic", "test", nil))
	<-started

	go func() {
		require.NoError(t, bus.Close())
		close(closed)
	}()

	select {
	case <-closed:
		t.Fatal("Close returned before in-flight handler completed")
	case <-time.After(25 * time.Millisecond):
	}

	close(release)

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Close")
	}
}

func TestLocalEventBus_SubscribeAfterCloseIgnored(t *testing.T) {
	bus := NewLocalEventBus()
	require.NoError(t, bus.Close())

	bus.Subscribe("closed.topic", func(_ Event) error {
		t.Fatal("handler should not be registered after close")
		return nil
	})

	assert.Empty(t, bus.subscribers["closed.topic"])
}

func TestNewEventFromCtx_Metadata(t *testing.T) {
	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	require.NoError(t, err)
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	require.NoError(t, err)

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)
	ctx = context.WithValue(ctx, ctxkeys.CorrelationIDKey, "corr_123")

	evt := NewEventFromCtx(ctx, "test.topic", "test", nil)

	assert.Equal(t, traceID.String(), evt.TraceID)
	assert.Equal(t, "corr_123", evt.CorrelationID)
}

func TestLocalEventBus_WithErrorHandlerInvoked(t *testing.T) {
	bus := NewLocalEventBusWithConfig(LocalBusConfig{DLQBufferSize: 1}, RetryConfig{
		MaxRetries:     1,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     time.Millisecond,
	})
	defer func() { _ = bus.Close() }()

	done := make(chan error, 1)
	bus.WithErrorHandler(func(topic string, err error) {
		assert.Equal(t, "error.handler", topic)
		done <- err
	})
	bus.Subscribe("error.handler", func(_ Event) error {
		return errors.New("handler failed")
	})

	bus.Publish(NewEvent("error.handler", "test", nil))

	select {
	case err := <-done:
		require.Error(t, err)
		assert.Contains(t, err.Error(), "handler failed")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for custom error handler")
	}
}

func TestNewEventBus_RejectsUnimplementedOrUnknownProviders(t *testing.T) {
	_, err := NewEventBus(BusConfig{Provider: "kafka"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kafka event bus is not yet implemented")

	_, err = NewEventBus(BusConfig{Provider: "rabbitmq"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rabbitmq")

	_, err = NewEventBus(BusConfig{Provider: "unknown_provider"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown event bus provider")

	bus, err := NewEventBus(BusConfig{Provider: "local"})
	require.NoError(t, err)
	require.NotNil(t, bus)
	require.NoError(t, bus.Close())
}
