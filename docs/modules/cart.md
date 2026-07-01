---
title: "Cart Module"
description: "Architecture, flows, and integration guide for the AxCom Cart module — cart management, item enrichment, count endpoint, and guest cart merge."
sidebar_label: Cart
sidebar_position: 2
---

<DocBadge status="under-review" version="v0.1.0-alpha" />

# Cart Module

The Cart module manages the authenticated user's shopping cart and provides a lightweight item count endpoint. It stores minimal cart state in the repository and enriches cart items using product catalog data.

---

## Capabilities

- `GET /api/cart` — Retrieve the authenticated user's full enriched cart.
- `GET /api/cart/count` — Retrieve total item count (lightweight, for UI badges).
- `POST /api/cart` — Add a product variant to the cart.
- `PUT /api/cart/items/:variantId` — Update the quantity of an existing cart item.
- `DELETE /api/cart/items/:variantId` — Remove a variant from the cart.
- `DELETE /api/cart` — Clear the authenticated user's cart.
- `POST /api/cart/merge` — Merge a guest cart into an authenticated user's cart.

---

## Architecture

```mermaid
flowchart LR
    A[HTTP Request] --> B[Cart Controller]
    B --> C[Cart Service]
    C --> D[Cart Repository]
    C --> E[Catalog Service]
    D --> F[(Cart DB)]
    E --> G[(Catalog DB)]
    C --> H[Response]
```

- `Cart Controller` handles request auth, validation, and response encoding.
- `Cart Service` handles persistence, catalog enrichment, and cart count logic.
- `Cart Repository` stores raw carts as customer ID + variant quantities.
- `Catalog Service` resolves product metadata for cart items.

---

## Module Structure

| File/Dir        | Role                                                |
| :-------------- | :-------------------------------------------------- |
| `controller.go` | HTTP handlers and request validation                |
| `routes.go`     | Route registration for cart endpoints               |
| `service.go`    | Business logic, cart enrichment, count calculations |
| `repository.go` | Cart persistence contract and repository wiring     |
| `model.go`      | Domain models used by the cart repository           |
| `errors.go`     | Module-specific sentinel error definitions          |
| `dto/`          | Request and response DTOs for cart API payloads     |
| `merge/`        | Isolated guest-cart merge submodule                 |

---

## Data Flow

```mermaid
sequenceDiagram
    participant U as User
    participant H as Cart Handler
    participant S as Cart Service
    participant R as Cart Repo
    participant C as Catalog Service

    U->>H: GET /api/cart
    H->>S: GetCart(customerID)
    S->>R: GetByCustomerID(customerID)
    R-->>S: raw cart items
    S->>C: GetProductByVariantID(variantID)
    C-->>S: product details
    S-->>H: enriched cart response
    H-->>U: 200 OK + cart payload
```

---

## Database Design

```mermaid
erDiagram
    CARTS {
        string customer_id PK
        CartItem[] items
    }

    CartItem {
        string variant_id
        int quantity
    }
```

---

## DTOs

| DTO                 | Fields                                                                         |
| :------------------ | :----------------------------------------------------------------------------- |
| `AddItemRequest`    | `variant_id`, `quantity`                                                       |
| `UpdateItemRequest` | `quantity`                                                                     |
| `CartItemResponse`  | `name`, `sku`, `price`, `discounted_price`, `image_url`, `stock`, `attributes` |
| `CartResponse`      | `customer_id`, `items`                                                         |
| `CartCountResponse` | `count`                                                                        |

---

## Merge Submodule

The merge logic lives in `internal/core/cart/merge`, keeping guest-cart merge behavior isolated from normal cart operations.

| File                  | Role                             |
| :-------------------- | :------------------------------- |
| `merge/dto.go`        | Merge request/response payloads  |
| `merge/controller.go` | Guest cart merge handler         |
| `merge/service.go`    | Merge business logic             |
| `merge/routes.go`     | Dedicated `POST /api/cart/merge` |

---

## Usage

```go
cartService := cart.NewCartService(cartRepo, catalogService)
cartController := cart.NewController(cartService)
cart.RegisterRoutes(routerGroup, cartController)

// For guest-cart merging:
cartMergeService := merge.NewMergeService(cartService, cartRepo)
cartMergeController := merge.NewController(cartMergeService)
merge.RegisterRoutes(routerGroup, cartMergeController)
```
