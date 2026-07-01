// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package dto

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
)

// validate is a local validator instance isolated from any global pkg/validator.
var validate = validator.New()

func init() {
	_ = validate.RegisterValidation("strong_password", validatePassword)
}

func validatePassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	if len(password) < 8 {
		return false
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= '0' && r <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasDigit && hasSpecial
}

// formatError provides a simple way to return clean validation errors
func formatError(err error) error {
	if err == nil {
		return nil
	}
	var errs validator.ValidationErrors
	if errors.As(err, &errs) {
		var messages []string
		for _, e := range errs {
			//nolint:gocritic // Using switch instead of if-else chain for future scalability of validation tags
			switch e.Tag() {
			case "required":
				messages = append(messages, e.Field()+" is required")
			case "email":
				messages = append(messages, e.Field()+" must be a valid email address")
			case "strong_password":
				messages = append(messages, e.Field()+" must be at least 8 characters and contain at least one uppercase letter, one lowercase letter, one number, and one special character")
			default:
				messages = append(messages, e.Field()+" is invalid")
			}
		}
		return errors.New(strings.Join(messages, ", "))
	}
	return err
}

// AuthRequest defines the JSON payload schema for authentication requests.
type AuthRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,strong_password"`
	Role     string `json:"role"`
}

// Validate validates the AuthRequest payload.
func (r *AuthRequest) Validate() error {
	return formatError(validate.Struct(r))
}

// LogoutRequest defines the JSON payload schema for logout requests.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// Validate validates the LogoutRequest payload.
func (r *LogoutRequest) Validate() error {
	return formatError(validate.Struct(r))
}

// RefreshRequest defines the JSON payload schema for token refresh requests.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// Validate validates the RefreshRequest payload.
func (r *RefreshRequest) Validate() error {
	return formatError(validate.Struct(r))
}

// PasswordResetRequest defines the JSON payload schema for requesting password recovery.
type PasswordResetRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// Validate validates the PasswordResetRequest payload.
func (r *PasswordResetRequest) Validate() error {
	return formatError(validate.Struct(r))
}

// PasswordResetConfirmRequest defines the JSON payload schema for confirming a password change.
type PasswordResetConfirmRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,strong_password"`
}

// Validate validates the PasswordResetConfirmRequest payload.
func (r *PasswordResetConfirmRequest) Validate() error {
	return formatError(validate.Struct(r))
}
