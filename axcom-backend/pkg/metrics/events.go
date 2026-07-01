// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ---------------------------------------------------------------------------
// Event bus — publish metrics
// ---------------------------------------------------------------------------

// EventsPublishedTotal counts every event dispatched to the bus, labelled by
// topic and source service. Use this to track per-topic throughput.
var EventsPublishedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: ns,
		Name:      "events_published_total",
		Help:      "Total events published to the event bus, partitioned by topic and source.",
	},
	[]string{"topic", "source"},
)

// EventsPublishErrorsTotal counts failures during event serialization or
// broker publish. Only fires for RabbitMQ; local publish is always synchronous.
var EventsPublishErrorsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: ns,
		Name:      "events_publish_errors_total",
		Help:      "Total failures publishing an event to the broker, partitioned by topic.",
	},
	[]string{"topic"},
)

// ---------------------------------------------------------------------------
// Event bus — consume metrics
// ---------------------------------------------------------------------------

// EventsConsumedTotal counts every handler invocation that reached a terminal
// state (success or failure after exhausting retries), partitioned by topic and
// outcome. Divide failure by total to get the error rate per topic.
var EventsConsumedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: ns,
		Name:      "events_consumed_total",
		Help:      "Total event handler executions that reached a terminal state (success/failure), by topic and status.",
	},
	[]string{"topic", "status"}, // status: "success" | "failure"
)

// EventsHandlerDurationSeconds tracks the end-to-end execution time of a
// single handler invocation (including all retry attempts), by topic.
var EventsHandlerDurationSeconds = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: ns,
		Name:      "events_handler_duration_seconds",
		Help:      "End-to-end duration of an event handler invocation (all retry attempts) in seconds, by topic.",
		Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30},
	},
	[]string{"topic"},
)

// ---------------------------------------------------------------------------
// Event bus — retry metrics
// ---------------------------------------------------------------------------

// EventsRetriesTotal counts individual retry attempts across all backends.
// For RabbitMQ, tier matches the retry delay tier (1, 2, 3...).
// For local, tier is always "1" (single exponential backoff sequence).
var EventsRetriesTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: ns,
		Name:      "events_retries_total",
		Help:      "Total retry attempts for failed event handlers, by topic, backend, and retry tier.",
	},
	[]string{"topic", "backend", "tier"}, // backend: "local" | "rabbitmq"
)

// ---------------------------------------------------------------------------
// Event bus — dead letter queue metrics
// ---------------------------------------------------------------------------

// EventsDLQTotal counts events routed to a dead letter queue after exhausting
// all retries, partitioned by topic and backend.
var EventsDLQTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: ns,
		Name:      "events_dlq_total",
		Help:      "Total events routed to the dead letter queue after exhausting retries, by topic and backend.",
	},
	[]string{"topic", "backend"},
)

// EventsDLQDroppedTotal counts events silently dropped because the local
// in-process DLQ channel was full. Any value > 0 indicates data loss.
var EventsDLQDroppedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: ns,
		Name:      "events_dlq_dropped_total",
		Help:      "Total events dropped because the local in-memory DLQ buffer was full. Any > 0 indicates data loss.",
	},
	[]string{"topic"},
)

// ---------------------------------------------------------------------------
// Outbox relay metrics
// ---------------------------------------------------------------------------

// EventsOutboxRelayPublishedTotal counts the cumulative number of outbox
// records successfully dispatched to the event bus by the relay.
var EventsOutboxRelayPublishedTotal = promauto.NewCounter(
	prometheus.CounterOpts{
		Namespace: ns,
		Name:      "events_outbox_relay_published_total",
		Help:      "Cumulative events published by the outbox relay worker.",
	},
)

// EventsOutboxRelayErrorsTotal counts failures in the outbox relay poll cycle
// (fetch errors or mark-published errors).
var EventsOutboxRelayErrorsTotal = promauto.NewCounter(
	prometheus.CounterOpts{
		Namespace: ns,
		Name:      "events_outbox_relay_errors_total",
		Help:      "Total errors encountered by the outbox relay (fetch or mark-published failures).",
	},
)

// EventsOutboxPendingBatch is a gauge set to the number of unpublished outbox
// records fetched on each relay cycle. A consistently non-zero value indicates
// the relay is falling behind production rate.
var EventsOutboxPendingBatch = promauto.NewGauge(
	prometheus.GaugeOpts{
		Namespace: ns,
		Name:      "events_outbox_pending_batch",
		Help:      "Number of unpublished outbox records fetched in the most recent relay cycle. A sustained value > 0 means the relay is lagging.",
	},
)

// ---------------------------------------------------------------------------
// Idempotency / dedup metrics
// ---------------------------------------------------------------------------

// EventsDedupSkippedTotal counts events discarded by the idempotency middleware
// because their ID was already recorded in the dedup store, partitioned by topic.
var EventsDedupSkippedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: ns,
		Name:      "events_dedup_skipped_total",
		Help:      "Total duplicate events skipped by the idempotency middleware, by topic.",
	},
	[]string{"topic"},
)

// ---------------------------------------------------------------------------
// RabbitMQ connection metrics
// ---------------------------------------------------------------------------

// EventsRabbitMQReconnectsTotal counts successful reconnections to the RabbitMQ
// broker after a connection loss. Each increment represents one full reconnect cycle.
var EventsRabbitMQReconnectsTotal = promauto.NewCounter(
	prometheus.CounterOpts{
		Namespace: ns,
		Name:      "events_rabbitmq_reconnects_total",
		Help:      "Total successful RabbitMQ reconnections after a connection loss.",
	},
)
