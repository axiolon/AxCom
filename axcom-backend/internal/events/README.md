# Events Module

This directory implements the asynchronous, event-driven messaging infrastructure used by `ecom-engine` to decouple domain workflows and publish notifications between modules. 

The module supports two primary execution environments: **Local (in-process) development** and **RabbitMQ-backed distributed production** routing, both featuring robust retry mechanisms and Dead Letter Queues (DLQ). Additionally, it implements a transactional **Outbox Pattern** to guarantee at-least-once delivery, alongside consumer-side **idempotency middleware** to eliminate duplicate processing.

---

## 1. Local Setup (In-Process Dev)

For local development or lightweight environments where a external broker is not required, the event bus runs as an in-process, asynchronous pub-sub dispatcher (`LocalEventBus`).

### Configuration
To use the local event bus, set `provider` to `"local"` or leave it empty in your service config (or set the environment variables `EVENT_BUS_PROVIDER=local`):

```yaml
events:
  provider: "local"
  retry:
    max_retries: 3
    initial_backoff: 50ms
    max_backoff: 2s
  local:
    dlq_buffer_size: 100
```

### Component Architecture & Delivery Flow
The local event bus operates asynchronously. Subscribers register their handlers, and publishers submit events. The bus dispatches each subscriber handler in a separate goroutine.

```mermaid
graph TD
    Publisher[Publisher] -->|Publish| Bus[LocalEventBus]
    Bus -->|Goroutine per Subscriber| Worker1[Worker Goroutine 1]
    Bus -->|Goroutine per Subscriber| Worker2[Worker Goroutine 2]
    Worker1 -->|Execute| Handler1[EventHandler 1]
    Worker2 -->|Execute| Handler2[EventHandler 2]
    
    Handler1 -->|Success| Done[Completed]
    Handler1 -->|Failure / Panic| Retry[Retry Loop]
    Retry -->|Max Retries Exceeded| DLQ[(In-Memory Dead Letter Queue Channel)]
```

### In-Memory Retry Loop & DLQ Flow
- **Concurrency Isolation**: Each handler runs concurrently in its own goroutine so slow consumers don't block others.
- **Fail-Safety & Backoff**: If a handler returns an error or panics, it enters an exponential backoff loop (e.g. `50ms`, `100ms`, `200ms` up to `max_backoff`) for up to `max_retries` attempts.
- **In-Memory DLQ**: If all attempts fail, the event is routed to an internal channel-buffered DLQ (`DLQ()`) where it can be read by monitoring utilities or logged.
- **Graceful Shutdown**: Calling `Close()` stops accepting new events, waits for all active handler goroutines to finish executing, and finally closes the DLQ channel.

```mermaid
sequenceDiagram
    participant App as Application Code
    participant Bus as LocalEventBus
    participant Worker as Goroutine (per Handler)
    participant DLQ as DLQ Channel

    App->>Bus: Subscribe(topic, handler)
    Note over Bus: Appends handler to map[topic]

    App->>Bus: Publish(Event)
    Note over Bus: Spawns a goroutine for each subscribed handler

    par Per Handler Executed Asynchronously
        Bus-->>Worker: invokeHandlerWithRetry(handler, event)
        loop Up to maxRetries (default 3)
            Worker->>Worker: executeHandler(handler, event)
            alt Handler Succeeded (No Panic or Error)
                Note over Worker: Returns (Success)
            else Handler Failed (Returns error or Panicked)
                Note over Worker: Backs off exponentially & retries
            end
        end
        Note over Worker: Retries exhausted
        Worker->>DLQ: sendToDLQ(event)
    end
```

---

## 2. RabbitMQ Setup (Distributed Prod)

For production, the module provides a production-grade, highly resilient `RabbitMQEventBus` that integrates with a RabbitMQ broker. It features a broker-managed tiered retry topology, consumer channel isolation, and automated connection healing.

### Configuration
Configure your service to use RabbitMQ by defining the broker credentials, exchange parameters, and retry characteristics:

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

### Channel Isolation & Thread Safety
- **Publish Channel**: A single, dedicated `amqp.Channel` is used for all publishes. Access is synchronized using a `sync.Mutex` (`pubMu`).
- **Consumer Channels**: AMQP channels are not goroutine-safe. Therefore, each call to `Subscribe()` opens a **dedicated AMQP channel** and spawns its own consumption loop. This eliminates cross-goroutine channel race conditions.

### Broker-Managed Tiered Retry & DLQ Topology
Rather than sleeping consumer threads inside the application (which blocks other messages), we leverage RabbitMQ's built-in **Dead Letter Exchanges (DLX)** and **Queue-level TTLs** to achieve non-blocking, delayed retries.

#### Topology Setup
- **Main Queue**: `ecom.{topic}` (bound to main exchange `ecom_events` with routing key `{topic}`).
- **Retry Exchanges & Queues**: One per delay tier (e.g. `ecom_events.retry.1`, `ecom_events.retry.2`, etc.).
  - Queues are configured with `x-message-ttl` matching the delay (e.g., 5s, 30s, 120s).
  - Queues are declared with `x-dead-letter-exchange` routing back to the main exchange (`ecom_events`), and `x-dead-letter-routing-key` routing back to `{topic}`.
  - No active consumers listen on these retry queues. They function purely as message delay timers.
- **DLQ Exchange & Queue**: `ecom_events_dlq` (fanout) and `ecom.{topic}.dlq` (exhausted messages queue).

#### Retry Flow
1. A message is delivered from the main queue `ecom.{topic}`.
2. If the handler fails, the consumer reads the `x-retry-count` header.
3. If `retryCount < len(retryDelays)`:
   - The consumer publishes the message to the corresponding retry tier exchange (e.g., `ecom_events.retry.1`) with the header `x-retry-count` incremented.
   - The consumer acknowledges (`Ack`) the original message on the main queue.
   - The message resides in the retry queue until `x-message-ttl` expires.
   - When it expires, RabbitMQ automatically dead-letters it back to the main exchange, which routes it back to the main queue for another consumer attempt.
4. If `retryCount >= len(retryDelays)`:
   - The consumer publishes the message directly to the DLQ exchange (`ecom_events_dlq`).
   - The consumer acknowledges (`Ack`) the original message.

```
                  ┌───────────────────────┐
                  │   Publish (Producer)  │
                  └───────────┬───────────┘
                              │
                              ▼
                     Exchange: ecom_events
                              │
                    (Topic Routing Key)
                              │
                              ▼
               Queue: ecom.<topic> (Main Queue)  ◄──────────────────────────────────┐
                              │                                                     │
                      (Consumer Read)                                               │
                              │                                                     │
                              ▼                                                     │
                   EventHandler Executes                                            │
                              │                                                     │
                  ┌───────────┴───────────┐                                         │
                  │   Did it succeed?     │                                         │
                  └─────┬───────────┬─────┘                                         │
                    Yes │        No │                                               │
                        ▼           ▼                                               │
                     [ Ack ]   Get `x-retry-count`                                  │
                                    │                                               │
                       ┌────────────┴────────────┐                                  │
                       │   Count < Max Tiers?    │                                  │
                       └─────┬──────────────┬────┘                                  │
                         Yes │           No │                                       │
                             ▼              ▼                                       │
                    Increment header   [ Ack Original ]                             │
                    Publish to Retry   Publish to DLQ Exchange                      │
                    Exchange           (ecom_events_dlq)                            │
                    [ Ack Original ]        │                                       │
                             │              ▼                                       │
                             ▼          Queue: ecom.<topic>.dlq                     │
                Exchange: ecom_events.retry.<N>                                     │
                             │                                                      │
                             ▼                                                      │
                Queue: ecom.<topic>.retry.<N>                                       │
                - x-message-ttl: Delay (e.g. 5s)                                    │
                - x-dead-letter-exchange: ecom_events ──────────────────────────────┘
```

### Connection Healing & Reconnection Loop
If the TCP connection to RabbitMQ is dropped:
1. `conn.NotifyClose` fires a notification.
2. The background `reconnectLoop()` detects the failure and initiates a connection retry cycle with exponential backoff (starting at `1s`, doubling up to a maximum of `30s`).
3. Once the connection is re-established:
   - It re-opens the publish channel and declares exchanges.
   - It iterates through and replays all stored subscriptions (`b.subscriptions`) by opening new consumer channels and restarting their consumption loops.

---

## 3. Outbox Pattern Implementation

Publishing events directly to a message broker during a database transaction is an anti-pattern. If the database transaction rolls back after the event is published, the broker is left with an invalid event (a ghost publish). If the publish fails, the database saves the state but the downstream systems never know (data loss).

To achieve **dual-write consistency**, we implement the **Transactional Outbox Pattern**.

```
   ┌──────────────────────────────────────────────────────────┐
   │                  Database Transaction                    │
   │                                                          │
   │  1. Mutation Query (e.g., INSERT INTO orders)            │
   │  2. Store Event Query (INSERT INTO outbox)               │
   │                                                          │
   └──────────────────────────┬───────────────────────────────┘
                              │
                              ▼ (Commit)
                  ┌───────────────────────┐
                  │    Database Table     │
                  │  ┌─────────────────┐  │
                  │  │     outbox      │  │
                  │  └────────┬────────┘  │
                  └───────────┼───────────┘
                              │
                   (Poll: SKIP LOCKED)
                              │
                              ▼
                  ┌───────────────────────┐
                  │     OutboxRelay       │
                  └───────────┬───────────┘
                              │
                          (Publish)
                              ▼
                  ┌───────────────────────┐
                  │  Event Bus (RabbitMQ) │
                  └───────────────────────┘
```

### Outbox Record & Schema
Events are persisted as JSON-serialized payloads in the `outbox` table alongside standard envelope metadata:

```go
type OutboxRecord struct {
	ID            string
	Topic         string
	Source        string
	Payload       []byte // JSON-encoded event payload
	Version       int
	TraceID       string
	CorrelationID string
	CreatedAt     time.Time
	PublishedAt   *time.Time
}
```

#### SQL Schema (`migrations/postgres/009_events.sql`):
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

### The Outbox Repository
The `OutboxRepository` interface dictates how events are written, fetched, and marked as processed:

```go
type OutboxRepository interface {
	Store(ctx context.Context, events ...Event) error
	FetchUnsent(ctx context.Context, batchSize int) ([]OutboxRecord, error)
	MarkPublished(ctx context.Context, ids []string) error
}
```

#### Postgres Implementation (`PostgresOutboxRepository`)
- **Store**: marshals the event payload and inserts it. Since it uses the context-based connection pool adapter, if there is a transaction in `ctx`, the record is stored as part of that transaction.
- **FetchUnsent**: fetches up to `batchSize` records that have not been published. Crucially, it uses **`SELECT ... FOR UPDATE SKIP LOCKED`**. This prevents multiple replicas of the application from picking up and publishing the same outbox records concurrently.
- **MarkPublished**: updates `published_at` for the batches successfully routed.

### Outbox Relay
The `OutboxRelay` is a background worker running in its own polling loop:

1. Wakes up every `PollInterval` (configured globally, e.g. 5 seconds).
2. Calls `FetchUnsent(ctx, batchSize)`.
3. For each fetched record:
   - Reconstructs the `Event` structure from `OutboxRecord`.
   - Dispatches it to the active `EventBus` using `bus.Publish()`.
4. Gathers all published IDs and updates their database status using `MarkPublished()`.

---

## 4. Consumer-Side Idempotency

Because the outbox relay guarantees **at-least-once delivery** (e.g. if the relay crashes after publishing but before marking the database record as published), consumers must be prepared to handle duplicate events safely.

### The DedupStore Interface
We track processed events using a unique event deduplication store:

```go
type DedupStore interface {
	Exists(ctx context.Context, eventID string) (bool, error)
	Mark(ctx context.Context, eventID string) error
}
```

#### PostgreSQL Dedup Table:
```sql
CREATE TABLE processed_events (
    event_id     VARCHAR(255) PRIMARY KEY,
    topic        VARCHAR(100) NOT NULL,
    processed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### Idempotency Middleware (`WithIdempotency`)
The `WithIdempotency` middleware intercepts event handler execution. It queries the `DedupStore` using the incoming event's unique ID:

- **Already processed**: returns `nil` immediately, logging and discarding the duplicate event.
- **Not processed**: executes the core handler. If successful, writes the event ID to the `DedupStore` using `Mark()`.

```go
// Wrapping an event handler with idempotency middleware
idempotentHandler := events.WithIdempotency(dedupStore, func(ev events.Event) error {
    // Core event logic here...
    return nil
})
bus.Subscribe(events.OrderCreatedTopic, idempotentHandler)
```

---

## 5. Usage Guide & Patterns

### Defining Topics and Payloads
Define your constants and payload structs inside `events.go`.

```go
const OrderCreatedTopic = "order.created"

type OrderCreatedEventPayload struct {
	OrderID    string    `json:"order_id"`
	CustomerID string    `json:"customer_id"`
	Total      float64   `json:"total"`
	CreatedAt  time.Time `json:"created_at"`
}
```

### 1. Generating & Publishing (Transactional Outbox)
Always construct events using `NewEventFromCtx` to propagate OpenTelemetry span contexts (`TraceID` and `CorrelationID`). Ensure you write to the outbox repository within your database transaction block:

```go
func (s *OrderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error) {
    var order *Order
    
    err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
        // 1. Perform database mutations
        var err error
        order, err = s.repo.Save(txCtx, req.ToEntity())
        if err != nil {
            return err
        }
        
        // 2. Build type-safe payload and event envelope
        payload := events.OrderCreatedEventPayload{
            OrderID:    order.ID,
            CustomerID: order.CustomerID,
            Total:      order.Total,
            CreatedAt:  order.CreatedAt,
        }
        event := events.NewEventFromCtx(txCtx, events.OrderCreatedTopic, "orders", payload)
        
        // 3. Write event to outbox within the transaction
        if s.outbox != nil {
            return s.outbox.Store(txCtx, event)
        }
        
        // Fallback for local development when outbox is disabled
        s.bus.Publish(event)
        return nil
    })
    
    return order, err
}
```

### 2. Subscribing & Type Assertions
Register handlers for topics, wrap them in the `WithIdempotency` middleware, and assert payload structures safely using the generic helper `AsPayload[T]`:

```go
// Wire event consumer in service initialization
func RegisterConsumers(bus events.EventBus, dedup events.DedupStore) {
    
    handler := func(ev events.Event) error {
        // Safely extract type-safe payload
        payload, ok := events.AsPayload[events.OrderCreatedEventPayload](ev)
        if !ok {
            // Log/discard schema mismatch
            return nil
        }
        
        // Execute business logic
        logger.Info("Processing Order: %s for customer %s", payload.OrderID, payload.CustomerID)
        return nil
    }
    
    // Apply idempotency middleware to guarantee once-and-only-once execution semantics
    idempotentHandler := events.WithIdempotency(dedup, handler)
    
    bus.Subscribe(events.OrderCreatedTopic, idempotentHandler)
}
```
