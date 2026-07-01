# Catalog Module

Provides product catalog functionality through feature submodules: core products, variants, images, discounts, bulk operations, and reviews.

## Quick Links

- [Full Documentation](../../../../../docs/modules/catalog.md)
- [Tests](./tests.md)

## Directory Layout

| File/Dir | Role |
| :--- | :--- |
| `routes.go` | Aggregates and registers all feature routes |
| `features/core/` | Core product CRUD and catalog retrieval |
| `features/variants/` | Product variant management |
| `features/images/` | Image upload, retrieval, and presign flow |
| `features/discounts/` | Discount and pricing rules |
| `features/bulk/` | Bulk catalog operations |
| `features/reviews/` | Product reviews with optional guest support |