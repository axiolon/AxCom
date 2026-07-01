---
title: "Payments Module"
description: "Architecture, flows, and integration guide for the AxCom Payments module — payment intents, provider callbacks, refunds, and event publishing."
sidebar_label: Payments
sidebar_position: 6
---

<DocBadge status="under-review" version="v0.1.0-alpha" />

# Payments Module

The Payments module orchestrates transaction flows with external payment gateways, handles callbacks/webhooks, maintains audit state of payment intents, and manages administrative tasks such as listing and refunds.

---

## Overview

The payments subsystem provides abstraction over multiple third-party payment providers (e.g., Stripe, PayPal, PayHere). It facilitates:

- **Payment Intents**: Initiating payments for pending customer orders.
- **Provider Callbacks**: Processing asynchronous webhooks or return URLs to verify transaction results.
- **Admin Actions**: Refunding successful transactions and listing execution history.
- **Event-Driven Workflows**: Publishing `payment.succeeded` or `payment.failed` to notify downstream services (Inventory, Orders).

---

## Architecture

```mermaid
flowchart TD
  Client["Client (Web/Mobile)"] -->|Create Intent / Callback| Gateway["API Gateway / Router"]
  Gateway -->|Auth / Route| Ctrl[Payments Controller]
  Gateway -->|Admin Auth / Route| AdminCtrl[Admin Payments Controller]
  Ctrl --> Service[Payment Service]
  AdminCtrl --> Service

  Service --> Repo[(Payments DB Repository)]
  Service --> OrderFetcher[Order Fetcher Interface]
  Service --> Providers[Payment Providers Map]

  Providers --> Stripe[Stripe Provider]
  Providers --> PayPal[PayPal Provider]
  Providers --> PayHere[PayHere Provider]

  Service -->|Publish Events| EventBus["Event Bus / PubSub"]
```

---

## API Routes

### Customer Endpoints

| Method | Route                          | Description                                                    | Auth           |
| :----- | :----------------------------- | :------------------------------------------------------------- | :------------- |
| `POST` | `/payments/intent`             | Creates a transaction intent with a specified/default provider | Yes (Customer) |
| `POST` | `/payments/callback/:provider` | Receives status callbacks/webhooks from the payment gateway    | No (Public)    |

### Admin Endpoints

| Method | Route                    | Description                                  | Auth        |
| :----- | :----------------------- | :------------------------------------------- | :---------- |
| `GET`  | `/admin/payments`        | Lists all payment logs/transactions          | Yes (Admin) |
| `POST` | `/admin/payments/refund` | Initiates a refund for a specific paid order | Yes (Admin) |

---

## Data Model

| Status      | Description                         |
| :---------- | :---------------------------------- |
| `pending`   | Intent created, awaiting completion |
| `succeeded` | Successfully captured               |
| `failed`    | Failed to capture                   |
| `refunded`  | Refunded by admin                   |

---

## Key Flows

### Create Payment Intent

```mermaid
sequenceDiagram
  participant Client as Customer Client
  participant Ctrl as Payments Controller
  participant Svc as Payment Service
  participant Fetcher as Order Fetcher
  participant Prov as Payment Provider
  participant Repo as DB Repository

  Client->>Ctrl: POST /payments/intent {order_id, provider}
  Ctrl->>Svc: CreatePaymentIntent(orderID, customerID, provider)
  Svc->>Fetcher: GetOrderAmountAndStatus(orderID)
  Fetcher-->>Svc: order total, status (pending)
  Svc->>Prov: CreateIntent(amount, currency)
  Prov-->>Svc: PaymentIntent {id, status, redirect_url}
  Svc->>Repo: Create(Payment Record)
  Svc-->>Ctrl: Payment details
  Ctrl-->>Client: 200 OK (Payment & Redirect details)
```

### Process Payment Callback (Webhook)

```mermaid
sequenceDiagram
  participant Prov as Payment Provider (Webhook)
  participant Ctrl as Payments Controller
  participant Svc as Payment Service
  participant Repo as DB Repository
  participant Bus as Event Bus

  Prov->>Ctrl: POST /payments/callback/:provider {intent_id, success}
  Ctrl->>Svc: ConfirmPayment(provider, intentID, success)
  Svc->>Repo: GetByProviderIntentID(provider, intentID)
  Repo-->>Svc: Payment Record

  alt success is true
    Svc->>Prov: ConfirmIntent(intentID)
    Svc->>Repo: Update Status to 'succeeded'
    Svc->>Bus: Publish 'payment.succeeded' event
  else success is false
    Svc->>Repo: Update Status to 'failed'
    Svc->>Bus: Publish 'payment.failed' event
  end

  Svc-->>Ctrl: Payment details
  Ctrl-->>Prov: 200 OK (processed)
```

### Admin Refund

```mermaid
sequenceDiagram
  participant Admin as Admin UI
  participant Ctrl as Admin Payments Controller
  participant Svc as Payment Service
  participant Repo as DB Repository
  participant Prov as Payment Provider

  Admin->>Ctrl: POST /admin/payments/refund {order_id}
  Ctrl->>Svc: RefundPayment(orderID)
  Svc->>Repo: GetByOrderID(orderID)
  Repo-->>Svc: Payment Record (succeeded)
  Svc->>Prov: RefundIntent(intentID, amount)
  Prov-->>Svc: Success
  Svc->>Repo: Update Status to 'refunded'
  Svc-->>Ctrl: Payment details
  Ctrl-->>Admin: 200 OK
```

---

## Event Subscriptions

| Event               | Trigger               | Consumers                                                    |
| :------------------ | :-------------------- | :----------------------------------------------------------- |
| `payment.succeeded` | Confirmation succeeds | Orders module (mark paid), Inventory (finalize reservations) |
| `payment.failed`    | Payment fails         | Inventory (release reservations)                             |
