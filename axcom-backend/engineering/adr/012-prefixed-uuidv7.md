# ADR-012: Cryptographically Secure Prefixed UUIDv7 Identifier Scheme

**Date:** 2026-06-27  
**Status:** accepted

## Context
Entities within the ecom-engine need globally unique identifiers. However, using traditional identifiers introduces several drawbacks:
- **Auto-incrementing integers** expose business volume and are susceptible to resource enumeration attacks.
- **Standard UUIDv4** identifiers are completely random. This causes heavy B-Tree index fragmentation and page splits in relational databases (like PostgreSQL) under write-heavy workloads.
- **Raw UUID strings** are anonymous. Looking at a raw UUID in logs, traces, or API responses makes it impossible to know if it refers to an order, product, or user without querying the database first.

We need a high-performance, sequence-ordered identifier scheme that is self-describing and secure.

## Decision
1. **Adopt UUIDv7:** Standardize on UUIDv7 for ID generation. The first 48 bits encode the Unix millisecond timestamp, ensuring sequential ordering for database insertions, while the remaining bits are random to prevent enumeration.
2. **Apply Entity Prefixes:** Prepend a short, lowercase entity type identifier followed by an underscore (e.g. `usr_` for users, `ord_` for orders, `prd_` for products) to the string representation of the UUIDv7. 
3. **Database-Agnostic Storage Translation:** 
   - Public APIs, service layers, and logs always use the prefixed string representation (e.g. `prd_019023ab-cdef-7000-8000-000000000003`).
   - Implement conversion helpers `ToUUID` and `FromUUID` in the `pkg/idgen` package. The repository layer uses these helpers to strip prefixes and pack IDs into native binary `UUID` structures for Postgres storage optimizations, or stores them as strings for MongoDB collections.

## Alternatives Considered

| Option | Reason Rejected |
|--------|-----------------|
| Standard UUIDv4 | Non-sequential keys disrupt database indexing performance and degrade write throughput. |
| Non-prefixed UUIDv7 | Lacks self-describing context, increasing developer friction when debugging logs, message queues, and JSON payloads. |
| ULID | While similar to UUIDv7, ULID is not standard UUID-compatible out of the box, complicating integration with standard SQL/NoSQL databases and third-party tooling. |

## Why This Choice
UUIDv7 provides time-ordered monotonicity, which prevents index splits in PostgreSQL and yields SQL database insertion performance close to sequential integers. Adding type prefixes makes the API and debugging logs self-documenting and human-readable, reducing cognitive overhead for engineers.

## Tradeoffs
**Gains:**
* High-performance, time-ordered database indexing (no page splits).
* Resistance to enumeration attacks.
* Highly readable, self-describing identifiers in APIs and system logs.

**Accepts:**
* Small CPU overhead for string parsing and prefix stripping at the persistence boundary.

## Consequences
* All new database tables and collections must use the prefixed string format externally.
* Database columns in Postgres should be defined as `UUID` binary types, requiring repository layers to translate prefixed strings using `idgen.ToUUID()` and `idgen.FromUUID()`.
