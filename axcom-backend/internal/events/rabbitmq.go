// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"ecom-engine/pkg/logger"
	"ecom-engine/pkg/metrics"
)

const retryCountHeader = "x-retry-count"

// subscription records a topic+handler pair so consumers can be replayed after reconnect.
type subscription struct {
	topic   string
	handler EventHandler
}

// RabbitMQEventBus implements EventBus using RabbitMQ with broker-managed tiered retry
// queues and a dead-letter queue.
//
// Topology per Subscribe(topic):
//
//	Main queue:    <prefix>.<topic>              — active consumers
//	Retry tier N:  <prefix>.<topic>.retry.<N>    — no consumers, fixed queue-level TTL, DLX -> main exchange
//	DLQ:           <prefix>.<topic>.dlq          — exhausted messages
//
// Retry flow (consumer-routed, broker-delayed):
//  1. Handler fails -> consumer reads x-retry-count header.
//  2. count < len(retryDelays) -> Ack + publish to retry tier exchange with incremented header.
//  3. Retry queue TTL expires -> broker DLX routes back to main exchange -> consumer picks up again.
//  4. count >= len(retryDelays) -> Ack + publish to DLQ exchange.
//
// Each consumer goroutine owns a dedicated AMQP channel (channels are NOT goroutine-safe).
// Publish() uses a separate mutex-protected channel.
// A reconnect loop watches conn.NotifyClose and replays all subscriptions on reconnect.
type RabbitMQEventBus struct {
	cfg         RabbitMQConfig
	retryDelays []time.Duration

	mu      sync.RWMutex
	conn    *amqp.Connection
	pubCh   *amqp.Channel
	pubMu   sync.Mutex // protects pubCh
	closed  bool
	closeCh chan struct{}
	wg      sync.WaitGroup

	subsMu        sync.Mutex
	subscriptions []subscription
	consumerChs   []*amqp.Channel
}

func newRabbitMQEventBus(cfg RabbitMQConfig, _ RetryConfig) (EventBus, error) {
	if cfg.ExchangeType == "" {
		cfg.ExchangeType = "topic"
	}
	if cfg.ExchangeName == "" {
		cfg.ExchangeName = "ecom_events"
	}
	if cfg.QueueName == "" {
		cfg.QueueName = "ecom"
	}
	if cfg.DLQExchange == "" {
		cfg.DLQExchange = cfg.ExchangeName + "_dlq"
	}
	if cfg.PrefetchCount <= 0 {
		cfg.PrefetchCount = 10
	}

	retryDelays := cfg.RetryDelays
	if len(retryDelays) == 0 {
		retryDelays = defaultRetryDelays
	}

	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq: dial %s: %w", cfg.URL, err)
	}

	pubCh, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq: open publish channel: %w", err)
	}

	b := &RabbitMQEventBus{
		cfg:         cfg,
		retryDelays: retryDelays,
		conn:        conn,
		pubCh:       pubCh,
		closeCh:     make(chan struct{}),
	}

	if err := b.declareExchanges(pubCh); err != nil {
		_ = conn.Close()
		return nil, err
	}

	go b.reconnectLoop()

	logger.Info("RabbitMQ event bus connected: exchange=%s, %d retry tiers, prefetch=%d",
		cfg.ExchangeName, len(retryDelays), cfg.PrefetchCount)

	return b, nil
}

func (b *RabbitMQEventBus) declareExchanges(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(b.cfg.ExchangeName, b.cfg.ExchangeType, true, false, false, false, nil); err != nil {
		return fmt.Errorf("rabbitmq: declare main exchange %q: %w", b.cfg.ExchangeName, err)
	}
	for i := range b.retryDelays {
		name := fmt.Sprintf("%s.retry.%d", b.cfg.ExchangeName, i+1)
		if err := ch.ExchangeDeclare(name, b.cfg.ExchangeType, true, false, false, false, nil); err != nil {
			return fmt.Errorf("rabbitmq: declare retry exchange %q: %w", name, err)
		}
	}
	if err := ch.ExchangeDeclare(b.cfg.DLQExchange, "fanout", true, false, false, false, nil); err != nil {
		return fmt.Errorf("rabbitmq: declare DLQ exchange %q: %w", b.cfg.DLQExchange, err)
	}
	return nil
}

func (b *RabbitMQEventBus) Subscribe(topic string, handler EventHandler) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return
	}
	b.mu.RUnlock()

	b.subsMu.Lock()
	b.subscriptions = append(b.subscriptions, subscription{topic, handler})
	b.subsMu.Unlock()

	b.startConsumer(topic, handler)
}

func (b *RabbitMQEventBus) startConsumer(topic string, handler EventHandler) {
	b.mu.RLock()
	conn := b.conn
	b.mu.RUnlock()

	ch, err := conn.Channel()
	if err != nil {
		logger.Error("rabbitmq: open consumer channel for topic %s: %v", topic, err)
		return
	}
	if err = ch.Qos(b.cfg.PrefetchCount, 0, false); err != nil {
		logger.Error("rabbitmq: set QoS for topic %s: %v", topic, err)
		_ = ch.Close()
		return
	}

	b.subsMu.Lock()
	b.consumerChs = append(b.consumerChs, ch)
	b.subsMu.Unlock()

	prefix := b.cfg.QueueName
	mainQueue := prefix + "." + topic

	if _, err = ch.QueueDeclare(mainQueue, true, false, false, false, nil); err != nil {
		logger.Error("rabbitmq: declare main queue %s: %v", mainQueue, err)
		return
	}
	if err = ch.QueueBind(mainQueue, topic, b.cfg.ExchangeName, false, nil); err != nil {
		logger.Error("rabbitmq: bind main queue %s: %v", mainQueue, err)
		return
	}

	for i, delay := range b.retryDelays {
		tierNum := i + 1
		retryExchange := fmt.Sprintf("%s.retry.%d", b.cfg.ExchangeName, tierNum)
		retryQueue := fmt.Sprintf("%s.%s.retry.%d", prefix, topic, tierNum)

		args := amqp.Table{
			"x-message-ttl":             int64(delay.Milliseconds()),
			"x-dead-letter-exchange":    b.cfg.ExchangeName,
			"x-dead-letter-routing-key": topic,
		}
		if _, err = ch.QueueDeclare(retryQueue, true, false, false, false, args); err != nil {
			logger.Error("rabbitmq: declare retry queue %s: %v", retryQueue, err)
			return
		}
		if err = ch.QueueBind(retryQueue, topic, retryExchange, false, nil); err != nil {
			logger.Error("rabbitmq: bind retry queue %s: %v", retryQueue, err)
			return
		}
	}

	dlqQueue := prefix + "." + topic + ".dlq"
	if _, err = ch.QueueDeclare(dlqQueue, true, false, false, false, nil); err != nil {
		logger.Error("rabbitmq: declare DLQ queue %s: %v", dlqQueue, err)
		return
	}
	if err = ch.QueueBind(dlqQueue, "", b.cfg.DLQExchange, false, nil); err != nil {
		logger.Error("rabbitmq: bind DLQ queue %s: %v", dlqQueue, err)
		return
	}

	deliveries, err := ch.Consume(mainQueue, "", false, false, false, false, nil)
	if err != nil {
		logger.Error("rabbitmq: consume %s: %v", mainQueue, err)
		return
	}

	b.wg.Add(1)
	go b.consumeLoop(ch, deliveries, handler, topic)
}

func (b *RabbitMQEventBus) Publish(event Event) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		logger.Warn("rabbitmq: event %s for topic %s dropped — bus is closed", event.ID, event.Topic)
		return
	}
	b.mu.RUnlock()

	body, err := json.Marshal(event)
	if err != nil {
		logger.Error("rabbitmq: marshal event %s: %v", event.ID, err)
		return
	}

	b.pubMu.Lock()
	defer b.pubMu.Unlock()

	err = b.pubCh.PublishWithContext(
		context.Background(),
		b.cfg.ExchangeName,
		event.Topic,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    event.ID,
			Timestamp:    event.Timestamp,
			Body:         body,
		},
	)
	if err != nil {
		logger.Error("rabbitmq: publish event %s (topic %s): %v", event.ID, event.Topic, err)
		metrics.EventsPublishErrorsTotal.WithLabelValues(event.Topic).Inc()
		return
	}
	metrics.EventsPublishedTotal.WithLabelValues(event.Topic, event.Source).Inc()
}

func (b *RabbitMQEventBus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	close(b.closeCh)
	b.mu.Unlock()

	b.wg.Wait()

	b.subsMu.Lock()
	for _, ch := range b.consumerChs {
		_ = ch.Close()
	}
	b.consumerChs = nil
	b.subsMu.Unlock()

	b.pubMu.Lock()
	_ = b.pubCh.Close()
	b.pubMu.Unlock()

	return b.conn.Close()
}

func (b *RabbitMQEventBus) consumeLoop(ch *amqp.Channel, deliveries <-chan amqp.Delivery, handler EventHandler, topic string) {
	defer b.wg.Done()
	for {
		select {
		case d, ok := <-deliveries:
			if !ok {
				return
			}
			b.handleDelivery(ch, d, handler, topic)
		case <-b.closeCh:
			return
		}
	}
}

func (b *RabbitMQEventBus) handleDelivery(ch *amqp.Channel, d amqp.Delivery, handler EventHandler, topic string) {
	var event Event
	if err := json.Unmarshal(d.Body, &event); err != nil {
		logger.Error("rabbitmq: unmarshal delivery on topic %s: %v — discarding", topic, err)
		_ = d.Ack(false)
		return
	}

	start := time.Now()
	err := executeRabbitHandler(handler, event)
	if err == nil {
		metrics.EventsConsumedTotal.WithLabelValues(topic, "success").Inc()
		metrics.EventsHandlerDurationSeconds.WithLabelValues(topic).Observe(time.Since(start).Seconds())
		_ = d.Ack(false)
		return
	}

	retryCount := getRetryCount(d.Headers)
	maxRetries := len(b.retryDelays)

	if retryCount < maxRetries {
		tierNum := retryCount + 1
		retryExchange := fmt.Sprintf("%s.retry.%d", b.cfg.ExchangeName, tierNum)

		logger.Warn("rabbitmq: event %s (topic %s) failed (attempt %d/%d), routing to retry tier %d: %v",
			event.ID, topic, retryCount+1, maxRetries, tierNum, err)
		metrics.EventsRetriesTotal.WithLabelValues(topic, "rabbitmq", strconv.Itoa(tierNum)).Inc()

		pubErr := ch.PublishWithContext(
			context.Background(),
			retryExchange,
			topic,
			false,
			false,
			amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				MessageId:    event.ID,
				Timestamp:    event.Timestamp,
				Headers:      amqp.Table{retryCountHeader: int64(tierNum)},
				Body:         d.Body,
			},
		)
		if pubErr != nil {
			logger.Error("rabbitmq: failed to publish event %s to retry tier %d: %v", event.ID, tierNum, pubErr)
		}
		_ = d.Ack(false)
		return
	}

	logger.Error("rabbitmq: event %s (topic %s) failed after %d retries, routing to DLQ: %v",
		event.ID, topic, maxRetries, err)
	metrics.EventsConsumedTotal.WithLabelValues(topic, "failure").Inc()
	metrics.EventsHandlerDurationSeconds.WithLabelValues(topic).Observe(time.Since(start).Seconds())
	metrics.EventsDLQTotal.WithLabelValues(topic, "rabbitmq").Inc()

	pubErr := ch.PublishWithContext(
		context.Background(),
		b.cfg.DLQExchange,
		topic,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    event.ID,
			Timestamp:    event.Timestamp,
			Headers:      d.Headers,
			Body:         d.Body,
		},
	)
	if pubErr != nil {
		logger.Error("rabbitmq: failed to publish event %s to DLQ: %v", event.ID, pubErr)
	}
	_ = d.Ack(false)
}

func (b *RabbitMQEventBus) reconnectLoop() {
	for {
		b.mu.RLock()
		conn := b.conn
		b.mu.RUnlock()

		notifyCh := conn.NotifyClose(make(chan *amqp.Error, 1))

		select {
		case amqpErr := <-notifyCh:
			if b.isClosed() {
				return
			}
			logger.Warn("rabbitmq: connection lost: %v, reconnecting...", amqpErr)
			b.reconnectWithBackoff()
		case <-b.closeCh:
			return
		}
	}
}

func (b *RabbitMQEventBus) reconnectWithBackoff() {
	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second

	for {
		if b.isClosed() {
			return
		}

		conn, err := amqp.Dial(b.cfg.URL)
		if err != nil {
			logger.Warn("rabbitmq: reconnect failed: %v, retrying in %s", err, backoff)
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		pubCh, err := conn.Channel()
		if err != nil {
			logger.Warn("rabbitmq: failed to open publish channel on reconnect: %v", err)
			_ = conn.Close()
			time.Sleep(backoff)
			continue
		}

		if err := b.declareExchanges(pubCh); err != nil {
			logger.Warn("rabbitmq: failed to declare exchanges on reconnect: %v", err)
			_ = conn.Close()
			time.Sleep(backoff)
			continue
		}

		b.mu.Lock()
		b.conn = conn
		b.mu.Unlock()

		b.pubMu.Lock()
		b.pubCh = pubCh
		b.pubMu.Unlock()

		b.subsMu.Lock()
		for _, oldCh := range b.consumerChs {
			_ = oldCh.Close()
		}
		b.consumerChs = nil
		subs := make([]subscription, len(b.subscriptions))
		copy(subs, b.subscriptions)
		b.subsMu.Unlock()

		b.wg.Wait()

		for _, sub := range subs {
			b.startConsumer(sub.topic, sub.handler)
		}

		metrics.EventsRabbitMQReconnectsTotal.Inc()
		logger.Info("rabbitmq: reconnected successfully, replayed %d subscription(s)", len(subs))
		return
	}
}

func (b *RabbitMQEventBus) isClosed() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.closed
}

func getRetryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}
	val, ok := headers[retryCountHeader]
	if !ok {
		return 0
	}
	switch v := val.(type) {
	case int64:
		return int(v)
	case int32:
		return int(v)
	case int:
		return v
	}
	return 0
}

func executeRabbitHandler(handler EventHandler, event Event) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return handler(event)
}

var _ EventBus = (*RabbitMQEventBus)(nil)
