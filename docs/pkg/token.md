---
title: token
sidebar_label: token
sidebar_position: 8
---

# token

<DocBadge status="under-review" version="v0.1.0-alpha" />

The `token` package provides two independent token mechanisms:

1. **`JWTManager`** — generates and validates internal HMAC-SHA256 signed tokens for service-to-service or session auth.
2. **`OIDCValidator`** — validates external JWT tokens (RS256/ES256) issued by a third-party Identity Provider (Auth0, Keycloak, Azure AD, etc.) using live JWKS key fetching.

**Import path:** `ecom-engine/pkg/token`

---

## When to use which

| | `JWTManager` | `OIDCValidator` |
|---|---|---|
| Token issuer | This service | External IdP |
| Signing algorithm | HMAC-SHA256 (symmetric) | RS256 / ES256 (asymmetric) |
| Key management | Single shared secret (`JWT_SECRET`) | JWKS URL fetched from IdP |
| Token format | Custom `base64url.payload.sig` | Standard RFC 7519 JWT |
| Use case | Internal session tokens, service auth | SSO, federated login |

---

## JWTManager

### Overview

`JWTManager` signs and validates tokens using HMAC-SHA256 with a shared secret. Tokens encode `userID`, `role`, and an expiry timestamp into a `base64url(payload).signature` format. This is **not** a standard RFC 7519 JWT — it will not be accepted by third-party JWT libraries.

### Setup

```go
import "ecom-engine/pkg/token"

mgr := token.NewJWTManager(os.Getenv("JWT_SECRET"))
```

### Claims struct

```go
type Claims struct {
    UserID string `json:"user_id"`
    Role   string `json:"role"`
    Exp    int64  `json:"exp"` // Unix timestamp
}
```

### Generate

```go
func (m *JWTManager) Generate(userID string, role string, duration time.Duration) (string, error)
```

Creates a signed token valid for `duration` from now.

```go
tok, err := mgr.Generate("usr_abc", "admin", 24*time.Hour)
// tok → "eyJ1c2VyX2lkIjoiLi4uIn0.xyz..."
```

### Validate

```go
func (m *JWTManager) Validate(tokenString string) (*Claims, error)
```

Verifies the signature using constant-time comparison (timing-safe), then checks expiry. Returns `*Claims` on success, or an error if the token is invalid, malformed, or expired.

```go
claims, err := mgr.Validate(tok)
if err != nil {
    // token invalid or expired
}
// claims.UserID → "usr_abc"
// claims.Role   → "admin"
```

### Security notes

- Signature comparison uses `crypto/subtle.ConstantTimeCompare` to prevent timing attacks.
- The signing key is held as `[]byte` in memory — never logged or serialised.
- Use a minimum 32-byte random secret for `JWT_SECRET`.

---

## OIDCValidator

### Overview

`OIDCValidator` validates standard JWT tokens (RS256/ES256) issued by an external Identity Provider. It fetches the provider's JWKS keys automatically and keeps them refreshed in the background using [`MicahParks/keyfunc`](https://github.com/MicahParks/keyfunc).

### Setup

```go
import "ecom-engine/pkg/token"

validator, err := token.NewOIDCValidator(
    "https://your-idp.auth0.com/",          // issuer
    "https://api.yourapp.com",              // audience
    "https://your-idp.auth0.com/.well-known/jwks.json", // JWKS URL
)
if err != nil {
    log.Fatal("failed to init OIDC validator:", err)
}
```

All three parameters are required. The JWKS keys are fetched immediately on construction — the call fails fast if the IdP is unreachable.

### OIDCClaims struct

```go
type OIDCClaims struct {
    Subject string   // "sub" — the IdP user identifier
    Email   string   // "email"
    Name    string   // "name"
    Roles   []string // "roles" — custom claim
}
```

### Validate

```go
func (v *OIDCValidator) Validate(tokenString string) (*OIDCClaims, error)
```

Validates the token against cached JWKS keys. Checks:

1. Signature (RS256/ES256 via JWKS)
2. `iss` (issuer) matches configured issuer
3. `aud` (audience) contains the configured audience
4. `exp` (expiry) is in the future
5. `sub` (subject) is non-empty

Returns `*OIDCClaims` on success, or a descriptive error indicating which check failed.

```go
claims, err := validator.Validate(bearerToken)
if err != nil {
    response.GinWriteError(c, apperrors.NewUnauthorized("invalid token", err))
    return
}
// claims.Subject → IdP user ID
// claims.Email   → user@example.com
// claims.Roles   → ["admin", "user"]
```

### JWKS key refresh

Keys are refreshed automatically in the background by the `keyfunc` library. This means key rotation at the IdP is handled without service restarts. The initial fetch happens at construction time (in `NewOIDCValidator`).
