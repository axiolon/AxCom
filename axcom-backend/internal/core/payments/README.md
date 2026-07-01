# Payments Module

Orchestrates transaction flows with external payment gateways (Stripe, PayPal, PayHere), handles callbacks/webhooks, manages payment intent state, and publishes payment events.

## Quick Links

- [Full Documentation](../../../../../docs/modules/payments.md)
- [Tests](./tests.md)

## Directory Layout

| File/Dir | Role |
| :--- | :--- |
| `controller.go` | Customer-facing HTTP handlers |
| `admin/` | Admin payment handlers (list, refund) |
| `service.go` | Business logic and provider orchestration |
| `repository.go` | Payment persistence contract |
| `model.go` | Payment intent and status models |
| `errors.go` | Payment-specific error definitions |
| `providers/` | Provider implementations (Stripe, PayPal, PayHere) |