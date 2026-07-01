---
title: validator
sidebar_label: validator
sidebar_position: 9
---

# validator

<DocBadge status="under-review" version="v0.1.0-alpha" />

The `validator` package provides common input validation helpers for request payloads. It wraps `go-playground/validator/v10` for struct validation and provides standalone functions for email and password rules.

**Import path:** `ecom-engine/pkg/validator`

---

## Functions

### ValidateStruct

```go
func ValidateStruct(s interface{}) error
```

Validates a struct's fields using `go-playground/validator/v10` struct tags. Returns `nil` if all rules pass, or a `validator.ValidationErrors` describing each failed field.

```go
type CreateProductRequest struct {
    Name  string  `validate:"required,min=1,max=200"`
    Price float64 `validate:"required,gt=0"`
    SKU   string  `validate:"required,alphanum"`
}

req := CreateProductRequest{Name: "", Price: -1}
if err := validator.ValidateStruct(req); err != nil {
    // err contains field-level validation errors
    response.GinWriteError(c, apperrors.NewBadRequest("invalid request", err))
    return
}
```

Common struct tag rules:

| Tag | Description |
|---|---|
| `required` | Field must be non-zero |
| `min=N` | Minimum length (strings) or value (numbers) |
| `max=N` | Maximum length or value |
| `email` | Valid email format |
| `gt=0` | Greater than zero |
| `alphanum` | Alphanumeric characters only |
| `oneof=a b c` | Value must be one of the listed options |
| `url` | Valid URL |

For the full list, see the [go-playground/validator documentation](https://pkg.go.dev/github.com/go-playground/validator/v10).

---

### ValidateEmail

```go
func ValidateEmail(email string) error
```

Validates an email address using Go's standard `net/mail.ParseAddress`, then additionally checks that the domain part contains a `.` (dot). Returns `errors.New("invalid email address")` on failure.

Rules applied:
- Parseable by RFC 5322
- Contains `@`
- Domain portion contains `.`

```go
if err := validator.ValidateEmail("user@example.com"); err != nil {
    return apperrors.NewBadRequest("invalid email", err)
}
```

---

### ValidatePassword

```go
func ValidatePassword(password string) error
```

Checks that the password meets minimum strength requirements.

| Rule | Detail |
|---|---|
| Minimum length | At least 8 characters |
| Letter required | At least one Unicode letter |
| Digit required | At least one Unicode digit |
| Special character required | At least one character that is not a letter, digit, or whitespace |

```go
if err := validator.ValidatePassword(pw); err != nil {
    return apperrors.NewBadRequest(err.Error(), nil)
}
```

Error messages returned:
- `"password must be at least 8 characters"`
- `"password must contain at least one letter, one number, and one special character"`

---

## Combining validators

For most handler request structs, use `ValidateStruct` first (covers field presence and format), then call `ValidateEmail` / `ValidatePassword` for the fields that need the stricter custom rules:

```go
type RegisterRequest struct {
    Email    string `json:"email"    validate:"required"`
    Password string `json:"password" validate:"required"`
    Name     string `json:"name"     validate:"required,min=1,max=100"`
}

if err := validator.ValidateStruct(req); err != nil {
    response.GinWriteError(c, apperrors.NewBadRequest("invalid request body", err))
    return
}
if err := validator.ValidateEmail(req.Email); err != nil {
    response.GinWriteError(c, apperrors.NewBadRequest(err.Error(), nil))
    return
}
if err := validator.ValidatePassword(req.Password); err != nil {
    response.GinWriteError(c, apperrors.NewBadRequest(err.Error(), nil))
    return
}
```
