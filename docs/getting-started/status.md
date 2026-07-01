---
title: "Project Status"
description: "Current development status of AxCom modules, infrastructure, and integrations."
sidebar_position: 2
---

# Project Status

<DocBadge status="under-review" version="v0.1.0-alpha" />

AxCom is in **early alpha**. The engine architecture is functional, but individual modules are at different stages of completeness. This page tracks what works, what is partially implemented, and what has not been tested yet.

:::caution Alpha Software
This project is not production-ready. APIs, data models, and configuration may change without notice between pre-release versions. Do not use this for real transactions or customer data without thorough independent testing.
:::

---

## Module Status

| Module | Status | Notes |
|:--|:--|:--|
| **Auth** | Under Review | Registration, login, JWT sessions, role-based access. Core flows implemented. |
| **Catalog** | Under Review | Products, variants, images, discounts, bulk ops, reviews. Core flows implemented. |
| **Cart** | Under Review | Cart management, item enrichment, guest merge. Core flows implemented. |
| **Inventory** | Under Review | Stock tracking, reservations, history, adjustments. Core flows implemented. |
| **Orders** | Under Review | Order creation, lifecycle state machine, guest checkout. Core flows implemented. |
| **Payments** | Under Review | Payment intents and refund logic exist. **No payment gateway has been integration-tested** (Stripe, PayPal, PayHere). Do not process real payments. |
| **Shipping** | Under Review | Rate calculation (flat rate, free-above, weight-based) implemented. **No real shipping provider connected or tested.** |

**Status definitions:**

- **Working** — Core functionality implemented, manually verified, and stable for development use.
- **Under Review** — Code exists and compiles, but has not been fully verified or tested end-to-end.
- **Partial** — Some features work, others are stubbed or incomplete.
- **Not Tested** — Code is written but has never been run against a real service or dataset.
- **Planned** — On the roadmap but no implementation yet.

---

## Infrastructure Status

| Component | Status | Notes |
|:--|:--|:--|
| **PostgreSQL** | Under Review | Schema migrations, repository layer, transactions. |
| **MongoDB** | Under Review | Indexes, repository layer, replica-set transactions. |
| **Redis Cache** | Under Review | Cache adapter implemented. |
| **In-Memory Cache** | Under Review | Works for local development without external dependencies. |
| **Local Event Bus** | Under Review | In-process pub/sub for development. |
| **RabbitMQ Events** | Under Review | Adapter exists. Not integration-tested at scale. |
| **Kafka Events** | Under Review | Adapter exists. Not integration-tested at scale. |
| **S3/R2 Storage** | Under Review | File upload adapter. Not tested against live buckets. |
| **Local Storage** | Under Review | File upload to local disk for development. |

---

## Integration Status

These are third-party integrations that require external accounts and credentials. None have been tested against live services yet.

| Integration | Provider | Status | Notes |
|:--|:--|:--|:--|
| Payment Gateway | Stripe | Not Tested | Adapter written, no live or sandbox testing done. |
| Payment Gateway | PayPal | Not Tested | Adapter written, no live or sandbox testing done. |
| Payment Gateway | PayHere | Not Tested | Adapter written, no live or sandbox testing done. |
| Object Storage | AWS S3 | Not Tested | Adapter written, not tested against real buckets. |
| Object Storage | Cloudflare R2 | Not Tested | Adapter written, not tested against real buckets. |
| Message Broker | RabbitMQ | Not Tested | Adapter written, not integration-tested. |
| Message Broker | Kafka | Not Tested | Adapter written, not integration-tested. |

---

## Test Coverage

| Area | Unit Tests | Integration Tests | E2E Tests |
|:--|:--|:--|:--|
| Auth | Under Review | Under Review | Under Review |
| Catalog | Under Review | Under Review | Under Review |
| Cart | Under Review | Under Review | Under Review |
| Inventory | Under Review | Under Review | Under Review |
| Orders | Under Review | Under Review | Under Review |
| Payments | Under Review | Under Review | Under Review |
| Shipping | Under Review | Under Review | Under Review |
| Engine / DI | Under Review | Under Review | — |
| Migrations | Under Review | Under Review | — |

---

## Updating This Page

As modules are verified and tested, update the status in this table. Use the status definitions above to keep things consistent. When a module moves to **Working**, note the version and date it was verified:

```markdown
| **Auth** | Working | Verified in v0.1.0-alpha (2026-07). Registration, login, JWT refresh all passing. |
```

This page is the single source of truth for what is safe to rely on.
