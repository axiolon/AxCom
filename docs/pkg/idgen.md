---
title: idgen
sidebar_label: idgen
sidebar_position: 4
---

# idgen

<DocBadge status="under-review" version="v0.1.0-alpha" />

The `idgen` package generates cryptographically secure, time-ordered, prefixed IDs built on **UUIDv7**. IDs are monotonically increasing within the same millisecond, making them safe to use as primary keys in sorted databases without hot-spotting.

**Import path:** `ecom-engine/pkg/idgen`

---

## ID format

```
<prefix><uuidv7>

Examples:
  usr_019123ab-cdef-7000-8000-000000000001
  ord_019123ab-cdef-7000-8000-000000000002
  evt_019123ab-cdef-7000-8000-000000000003
```

The prefix is a short lowercase string followed by an underscore (`_`). It makes IDs self-describing - you can identify the entity type at a glance without querying the database.

---

## Functions

### Generate

```go
func Generate(prefix string) (string, error)
```

Creates a new UUIDv7 and prepends `prefix`. Returns an error if the OS CSPRNG is unavailable (should be extremely rare).

```go
id, err := idgen.Generate("usr_")
// id → "usr_019123ab-cdef-7000-8000-000000000001"
```

### MustGenerate

```go
func MustGenerate(prefix string) string
```

Like `Generate` but panics on error. Use in initialization code or tests where error handling would be noise.

```go
id := idgen.MustGenerate("evt_")
```

### ToUUID

```go
func ToUUID(prefixedID string) (uuid.UUID, error)
```

Strips the prefix and parses the remaining string as a `uuid.UUID`. Returns `uuid.Nil` and an error if the UUID portion is malformed. If no underscore is found, it attempts to parse the full string as a UUID directly.

Intended for the **repository layer** when storing IDs in columns that use UUID or BINARY(16) types.

```go
rawUUID, err := idgen.ToUUID("ord_019123ab-cdef-7000-8000-000000000001")
// rawUUID → uuid.UUID{...}
```

### MustToUUID

```go
func MustToUUID(prefixedID string) uuid.UUID
```

Like `ToUUID` but panics on error.

```go
rawUUID := idgen.MustToUUID("ord_019123ab-cdef-7000-8000-000000000001")
```

### FromUUID

```go
func FromUUID(prefix string, id uuid.UUID) string
```

Re-attaches the prefix to a raw `uuid.UUID`. The inverse of `ToUUID`. Use when reading UUID primary keys from the DB and reconstructing the prefixed application ID.

```go
prefixedID := idgen.FromUUID("ord_", rawUUID)
// prefixedID → "ord_019123ab-cdef-7000-8000-000000000001"
```

---

## Prefix conventions

| Entity  | Prefix |
| ------- | ------ |
| User    | `usr_` |
| Order   | `ord_` |
| Product | `prd_` |
| Cart    | `crt_` |
| Event   | `evt_` |
| Session | `ses_` |

Define your module's prefix as a constant to avoid typos:

```go
const orderPrefix = "ord_"

id, err := idgen.Generate(orderPrefix)
```

---

## Why UUIDv7?

- **Time-ordered** - the first 48 bits encode the Unix millisecond timestamp, so rows inserted sequentially have monotonically increasing keys. This avoids B-tree page splits and hot-spotting in PostgreSQL.
- **Random suffix** - the remaining bits are random, preventing enumeration attacks.
- **Standard format** - compatible with any system that accepts UUID strings (Postgres `uuid`, MySQL `char(36)`, etc.).
