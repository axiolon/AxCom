// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"errors"
	"net/mail"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// ValidateStruct validates a struct's fields based on its struct tags.
func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}

// ValidateEmail checks if the string is a valid email address.
func ValidateEmail(email string) error {
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return errors.New("invalid email address")
	}
	parts := strings.Split(addr.Address, "@")
	if len(parts) != 2 || !strings.Contains(parts[1], ".") {
		return errors.New("invalid email address")
	}
	return nil
}

// ValidatePassword checks if the password meets the strength requirements:
// at least 8 characters, at least one letter, one number, and one special character.
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	var hasLetter, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		case !unicode.IsSpace(r):
			hasSpecial = true
		}
	}
	if !hasLetter || !hasDigit || !hasSpecial {
		return errors.New("password must contain at least one letter, one number, and one special character")
	}
	return nil
}
