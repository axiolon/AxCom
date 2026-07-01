# Architecture Decision Records (ADRs)

Lightweight records of significant architectural decisions made in this project.

## Convention

- **Naming:** `NNN-short-title.md` (e.g., `001-error-handling.md`)
- **Numbering:** Sequential, never reused
- **Status:** `accepted` | `superseded by NNN` | `deprecated`

## Index

| #   | Title                              | Status   |
|-----|------------------------------------|----------|
| 001 | [Error Handling](001-error-handling.md) | accepted |
| 002 | [Transaction Management](002-transaction-management.md) | accepted |
| 003 | [Repository Deduplication](003-repository-deduplication.md) | accepted |
| 004 | [Query Safety](004-query-safety.md) | accepted |
| 005 | [Security Headers Middleware Configuration](005-security-middleware.md) | accepted |
| 006 | [Rate Limiting Strategy and Implementation](006-rate-limiting.md) | accepted |
| 007 | [Distributed Rate Limiting and Fail-Open Failover Architecture](007-rate-limiting-failover.md) | accepted |
| 008 | [Modular Monolith and Kahn's Dependency Resolution](008-modular-monolith.md) | accepted |
| 009 | [Hand-rolled Map-Backed Dependency Injection Container](009-dependency-injection.md) | accepted |
| 010 | [Database-Agnostic RepoProvider Factory Pattern](010-repo-provider.md) | accepted |
| 011 | [Unified Database Migrations and Seeding CLI](011-database-migrations.md) | accepted |
| 012 | [Cryptographically Secure Prefixed UUIDv7 Identifier Scheme](012-prefixed-uuidv7.md) | accepted |
| 013 | [Domain-Aware Application Error Architecture](013-app-errors.md) | accepted |
| 014 | [Observability and Telemetry Architecture](014-observability-telemetry.md) | accepted |
| 015 | [Customized HMAC-SHA256 Signed Security Token Authentication](015-hmac-tokens.md) | accepted |

