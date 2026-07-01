# ADR-003: Feature-Isolated Repositories

**Status:** accepted

## Context

The inventory module has 8 feature sub-packages (core, bulk, history, reservation, reports, transfer, adjustment, sync). Five of them (adjustment, bulk, sync, transfer, reservation) contain identical `GetStock` and `SaveStock` implementations across both Postgres and MongoDB backends (10 files total).

A shared `StockBase` struct via Go embedding was considered to eliminate this duplication.

## Decision

**Keep each feature repository self-contained with its own `GetStock`/`SaveStock` copy.**

### Why intentional duplication over shared code?

- **Feature isolation** — Each feature package (adjustment, transfer, reservation, etc.) is fully decoupled. You can delete an entire feature folder and nothing else breaks. A shared `StockBase` creates a dependency that couples all features together.
- **Independent evolution** — If a feature later needs a different query (e.g., `SELECT ... FOR UPDATE` in transfer for row-level locking, or additional columns in reservation), it can diverge without affecting other features.
- **Removability** — The module is designed so features can be added or removed without cascading changes. A shared base violates this by making every feature depend on a central package.

### Trade-off acknowledged

Bug fixes to `GetStock`/`SaveStock` must be applied to each copy. This is mitigated by:

1. These methods are simple (single query, scan, return) and unlikely to change often.
2. The ADR-001 and ADR-004 fixes (domain errors, `rows.Err()` checks) are applied uniformly across all copies as a one-time hardening pass.
3. The interface contracts in the core layer catch any drift at compile time.

## Consequences

- Each feature package remains independently deployable and removable.
- No cross-feature coupling at the repository layer.
- Developers must remember to apply shared query changes to all copies (low frequency, caught by tests).
