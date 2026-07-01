---
title: "Event-Driven Messaging"
description: "Async pub/sub infrastructure with local and RabbitMQ backends, transactional outbox pattern, tiered retries, dead letter queues, and consumer-side idempotency."
sidebar_position: 3
---

# Event-Driven Messaging Infrastructure

<DocBadge status="under-review" version="v0.1.0-alpha" />

The `internal/events` package implements the asynchronous messaging backbone for AxCom. It decouples domain workflows, propagates notifications between services, and guarantees at-least-once delivery through a transactional outbox pattern. Consumer-side idempotency middleware prevents duplicate side effects.

---

## Architecture Overview

```mermaid
flowchart TD
    BL["Business Logic\n(Service Layer)\n\n1. Mutation — INSERT INTO orders\n2. outbox.Store(ctx, event)"]
    OT[("outbox table\nPostgreSQL")]
    OR["OutboxRelay\nbackground goroutine"]
    EB["EventBus\nlocal · rabbitmq"]
    EH["EventHandler\n+ WithIdempotency"]

    BL -->|"atomic commit"| OT
    OT -->|"poll every 5s\nSELECT FOR UPDATE SKIP LOCKED"| OR
    OR -->|"bus.Publish(event)"| EB
    EB -->|"fan-out per topic"| EH
```

---

## 1. Local Event Bus (Development)

The `LocalEventBus` is an in-process, goroutine-based pub/sub dispatcher with no external dependencies. Use it during local development or in unit tests.

### Configuration

```yaml
events:
  provider: "local" # or omit — local is the default
  retry:
    max_retries: 3
    initial_backoff: 50ms
    max_backoff: 2s
  local:
    dlq_buffer_size: 100
```

### How It Works

- `Publish(event)` copies the handler slice and spawns one goroutine per registered handler.
- Each goroutine runs `invokeHandlerWithRetry`, which calls the handler up to `max_retries` times with exponential backoff (`initial_backoff` × 2ⁿ, capped at `max_backoff`).
- Panics inside handlers are recovered and converted to errors, which enter the same retry loop.
- After all retries are exhausted the event is pushed to an in-memory `chan Event` (the DLQ). If the channel is full the event is dropped — this will fire the `events_dlq_dropped_total` metric and a Grafana alert.
- `Close()` stops new publishes, drains all active handler goroutines via a `sync.WaitGroup`, then closes the DLQ channel.

```mermaid
sequenceDiagram
    participant App
    participant LocalEventBus
    participant Goroutine
    participant DLQ

    App->>LocalEventBus: Publish(event)
    LocalEventBus-->>Goroutine: invokeHandlerWithRetry()
    loop Up to max_retries
        Goroutine->>Goroutine: executeHandler()
        alt success
            Note over Goroutine: done ✓
        else error / panic
            Note over Goroutine: exponential backoff → retry
        end
    end
    Goroutine->>DLQ: sendToDLQ(event)  [if all retries fail]
```

### Reading the DLQ

```go
bus := events.NewLocalEventBus()
go func() {
    for evt := range bus.DLQ() {
        log.Printf("DLQ event: %s topic=%s", evt.ID, evt.Topic)
    }
}()
```

---

## 2. RabbitMQ Event Bus (Production)

The `RabbitMQEventBus` integrates with a RabbitMQ broker and delegates retry scheduling entirely to the broker using Dead Letter Exchanges (DLX) and per-queue TTLs. This keeps consumer goroutines non-blocking between retries.

### Configuration

```yaml
events:
  provider: "rabbitmq"
  rabbitmq:
    url: "amqp://guest:guest@localhost:5672/"
    exchange_name: "ecom_events"
    exchange_type: "topic"
    queue_name: "ecom"
    dlq_exchange: "ecom_events_dlq"
    prefetch_count: 10
    retry_delays:
      - 5s
      - 30s
      - 120s
```

The length of `retry_delays` sets the maximum number of retry tiers. Default is `[5s, 30s, 120s]` (3 tiers).

### Topology

For each `Subscribe(topic, handler)` call the bus declares:

| Resource         | Name pattern                  | Role                                |
| ---------------- | ----------------------------- | ----------------------------------- |
| Main queue       | `ecom.<topic>`                | Active consumer target              |
| Retry queue N    | `ecom.<topic>.retry.N`        | Delay buffer (TTL = retryDelays[N]) |
| DLQ queue        | `ecom.<topic>.dlq`            | Exhausted messages                  |
| Main exchange    | `ecom_events` (topic)         | Routes publishes to main queue      |
| Retry exchange N | `ecom_events.retry.N` (topic) | Routes retries into retry queue N   |
| DLQ exchange     | `ecom_events_dlq` (fanout)    | Routes DLQ messages                 |

Retry queues have no active consumers. They are configured with:

- `x-message-ttl` = delay in milliseconds
- `x-dead-letter-exchange` = `ecom_events` (the main exchange)
- `x-dead-letter-routing-key` = `<topic>`

When the TTL expires, the broker automatically dead-letters the message back to the main queue for re-delivery.

### Retry Flow

```mermaid
flowchart TD
    PUB["Publisher"]
    EX["Exchange: ecom_events\ntopic exchange"]
    MQ["Queue: ecom.&lt;topic&gt;\nmain queue"]
    HND["Handler executes"]
    ACK["Ack ✓"]
    RC{"retryCount\n< maxRetries?"}
    RX["Exchange: ecom_events.retry.N"]
    RQ["Queue: ecom.&lt;topic&gt;.retry.N\nx-message-ttl = delay\nx-dead-letter-exchange = ecom_events"]
    DLX["Exchange: ecom_events_dlq\nfanout"]
    DLQ["Queue: ecom.&lt;topic&gt;.dlq\nmanual inspection"]

    PUB -->|"Publish(event)"| EX
    EX -->|"routing key = topic"| MQ
    MQ -->|"consumer reads"| HND
    HND -->|"success"| ACK
    HND -->|"failure"| RC
    RC -->|"yes — increment x-retry-count"| RX
    RX --> RQ
    RQ -->|"TTL expires → DLX routes back"| MQ
    RC -->|"no — retries exhausted"| DLX
    DLX --> DLQ
```

### Channel Isolation & Thread Safety

- **Publish**: uses a single `amqp.Channel` protected by `sync.Mutex` (`pubMu`). AMQP channels are not goroutine-safe.
- **Consumers**: each `Subscribe` call opens a **dedicated channel** and a dedicated goroutine loop, fully isolating consumers from each other and from the publish channel.

### Connection Healing

If the TCP connection drops:

1. `conn.NotifyClose` fires in the background `reconnectLoop`.
2. The loop retries with exponential backoff (1s → 2s → 4s ... up to 30s).
3. On success: re-opens the publish channel, re-declares exchanges, and replays all registered `subscriptions` by restarting their consumer goroutines.
4. `events_rabbitmq_reconnects_total` increments on every successful reconnect.

---

## 3. Transactional Outbox Pattern

Publishing events directly inside a database transaction is unsafe:

- If the transaction **rolls back** after a publish, the broker holds a ghost event.
- If the **publish fails**, the database commits but downstream never learns.

The outbox pattern solves this with a two-phase write:

### Phase 1 — Write (inside your DB transaction)

```go
err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
    order, err := s.repo.Save(txCtx, req.ToEntity())
    if err != nil {
        return err
    }

    event := events.NewEventFromCtx(txCtx, events.OrderCreatedTopic, "orders",
        events.OrderCreatedEventPayload{
            OrderID:    order.ID,
            CustomerID: order.CustomerID,
            Total:      order.Total,
            CreatedAt:  order.CreatedAt,
        })

    return s.outbox.Store(txCtx, event)  // same transaction
})
```

Both the business write and the outbox write commit atomically. If either fails, nothing is published.

### Phase 2 — Relay (background polling)

`OutboxRelay` wakes up every `poll_interval` (default 5s) and:

1. Runs `SELECT ... FOR UPDATE SKIP LOCKED` for up to `batch_size` unpublished records. `SKIP LOCKED` prevents multiple relay instances from double-processing the same batch.
2. Calls `bus.Publish(event)` for each record.
3. Calls `MarkPublished(ids)` to set `published_at` for the batch.

If `MarkPublished` fails (e.g. DB connectivity lost after publish), the relay will re-fetch those records on the next cycle and publish them again — this is the **at-least-once** guarantee. Consumer idempotency (see §4) absorbs the duplicate.

### Outbox Schema

```sql
CREATE TABLE outbox (
    id              VARCHAR(255) PRIMARY KEY,
    topic           VARCHAR(100) NOT NULL,
    source          VARCHAR(100) NOT NULL,
    payload         JSONB NOT NULL,
    version         INTEGER NOT NULL DEFAULT 1,
    trace_id        VARCHAR(64),
    correlation_id  VARCHAR(64),
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at    TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_outbox_unpublished ON outbox(created_at) WHERE published_at IS NULL;
```

---

## 4. Consumer-Side Idempotency

Because the outbox relay guarantees **at-least-once** delivery, consumers must tolerate receiving the same event ID more than once.

The `WithIdempotency` middleware wraps any `EventHandler` and guards execution with a dedup store lookup:

```go
handler := func(ev events.Event) error {
    payload, ok := events.AsPayload[events.OrderCreatedEventPayload](ev)
    if !ok {
        return nil
    }
    return processOrder(payload)
}

idempotentHandler := events.WithIdempotency(dedupStore, handler)
bus.Subscribe(events.OrderCreatedTopic, idempotentHandler)
```

Execution flow:

```mermaid
flowchart TD
    EV["Incoming Event"]
    CHK{"dedupStore.Exists\nevent.ID ?"}
    SKIP["log + skip\nreturn nil\nevents_dedup_skipped_total++"]
    RUN["Run handler"]
    OK{"handler\nerror?"}
    MARK["dedupStore.Mark(event.ID)"]
    ERR["return error\n→ retry will re-attempt\nMark NOT called"]

    EV --> CHK
    CHK -->|"yes — duplicate"| SKIP
    CHK -->|"no — first time"| RUN
    RUN --> OK
    OK -->|"nil"| MARK
    OK -->|"err"| ERR
```

### Dedup Store Schema

```sql
CREATE TABLE processed_events (
    event_id     VARCHAR(255) PRIMARY KEY,
    topic        VARCHAR(100) NOT NULL,
    processed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

A Redis-backed `DedupStore` can replace the PostgreSQL implementation for lower-latency dedup lookups at the cost of TTL-based expiry (rather than permanent dedup).

---

## 5. Trace Context Propagation

Every event carries a `trace_id` and `correlation_id` in its envelope. These are extracted from the OpenTelemetry span context at the time of publishing:

```go
event := events.NewEventFromCtx(ctx, topic, source, payload)
// event.TraceID       = active OTel span trace ID (if present)
// event.CorrelationID = ctx value at ctxkeys.CorrelationIDKey (if present)
```

Both fields are persisted in the `outbox` table and propagated through the relay, so log lines in consumers carry the same trace ID as the originating HTTP request. Use `trace_id` in Loki log queries to correlate the full lifecycle of a business transaction across service boundaries.

---

## 6. Defining Topics and Payloads

All topic constants and payload structs are co-located in `internal/events/events.go`:

```go
const OrderCreatedTopic = "order.created"

type OrderCreatedEventPayload struct {
    OrderID    string    `json:"order_id"`
    CustomerID string    `json:"customer_id"`
    Total      float64   `json:"total"`
    CreatedAt  time.Time `json:"created_at"`
}
```

Use the generic `AsPayload[T]` helper in handlers to perform a type-safe payload assertion:

```go
payload, ok := events.AsPayload[events.OrderCreatedEventPayload](ev)
if !ok {
    return nil  // schema mismatch — discard safely
}
```

---

## 7. Observability

### Prometheus Metrics (`pkg/metrics/events.go`)

All metrics are prefixed `ecom_engine_events_`.

| Metric                                | Type      | Labels                     | Description                                                      |
| ------------------------------------- | --------- | -------------------------- | ---------------------------------------------------------------- |
| `events_published_total`              | Counter   | `topic`, `source`          | Events dispatched to the bus                                     |
| `events_publish_errors_total`         | Counter   | `topic`                    | Broker publish failures (RabbitMQ)                               |
| `events_consumed_total`               | Counter   | `topic`, `status`          | Terminal handler executions (`success`/`failure`)                |
| `events_handler_duration_seconds`     | Histogram | `topic`                    | End-to-end handler latency (all retry attempts)                  |
| `events_retries_total`                | Counter   | `topic`, `backend`, `tier` | Individual retry attempts                                        |
| `events_dlq_total`                    | Counter   | `topic`, `backend`         | Events routed to DLQ after exhausting retries                    |
| `events_dlq_dropped_total`            | Counter   | `topic`                    | Events dropped because the local DLQ buffer was full (data loss) |
| `events_outbox_relay_published_total` | Counter   | —                          | Cumulative events dispatched by the relay                        |
| `events_outbox_relay_errors_total`    | Counter   | —                          | Relay poll or mark-published failures                            |
| `events_outbox_pending_batch`         | Gauge     | —                          | Unpublished records in the last relay cycle                      |
| `events_dedup_skipped_total`          | Counter   | `topic`                    | Duplicate events discarded by idempotency middleware             |
| `events_rabbitmq_reconnects_total`    | Counter   | —                          | Successful RabbitMQ reconnections                                |

### Key PromQL Queries

```promql
# Handler error rate (5m window)
sum(rate(ecom_engine_events_consumed_total{status="failure"}[5m]))
  / clamp_min(sum(rate(ecom_engine_events_consumed_total[5m])), 0.001)

# DLQ events in the last hour, by topic
sum by (topic, backend) (increase(ecom_engine_events_dlq_total[1h]))

# p99 handler latency
histogram_quantile(0.99,
  sum by (le) (rate(ecom_engine_events_handler_duration_seconds_bucket[5m])))

# Outbox lag (relay falling behind)
ecom_engine_events_outbox_pending_batch
```

### Grafana Dashboard

Import `deployments/monitoring/grafana/dashboards/ecom-engine/events.json`. The dashboard contains four sections:

| Section                    | Panels                                                                                         |
| -------------------------- | ---------------------------------------------------------------------------------------------- |
| **Publish & Consume**      | Publish rate, consume success/failure rates, error %, throughput by topic, latency percentiles |
| **Retry & DLQ**            | Retry rate by topic and tier, DLQ events over time, DLQ bar gauge by topic                     |
| **Outbox Relay**           | Relay throughput, error rate, pending batch size (stat + time-series)                          |
| **Idempotency & RabbitMQ** | Dedup skip rate, reconnection counter, publish errors                                          |

### Alerts (`deployments/monitoring/grafana/provisioning/alerting/events-alerts.yml`)

| Alert                       | Severity     | Trigger                      |
| --------------------------- | ------------ | ---------------------------- |
| Events Routed to DLQ        | **critical** | Any DLQ write in 5m          |
| Events Dropped (DLQ Full)   | **critical** | Any drop (data loss)         |
| High Handler Error Rate     | warning      | > 5% error rate for 5m       |
| Outbox Relay Errors         | warning      | Any relay error sustained 5m |
| Outbox Relay Backlog        | warning      | Pending batch ≥ 90 for 10m   |
| RabbitMQ Reconnection Spike | warning      | > 2 reconnects in 10m        |
| High Duplicate Event Rate   | warning      | > 1 dedup skip/s for 10m     |

---

## 8. Environment Comparison

| Feature             | Local Bus                                         | RabbitMQ Bus                              |
| ------------------- | ------------------------------------------------- | ----------------------------------------- |
| External dependency | None                                              | RabbitMQ broker                           |
| Retry mechanism     | In-process exponential backoff (sleeps goroutine) | Broker-managed TTL queues (non-blocking)  |
| DLQ                 | In-memory channel (`chan Event`)                  | Durable broker queue (`ecom.<topic>.dlq`) |
| DLQ persistence     | Lost on process restart                           | Durable across restarts                   |
| Multi-instance safe | No (single process)                               | Yes (SKIP LOCKED + broker routing)        |
| Connection healing  | N/A                                               | Automatic reconnect with backoff          |
| Recommended use     | Local dev, unit tests                             | Staging, production                       |
