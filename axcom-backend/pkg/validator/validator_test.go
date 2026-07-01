// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestUser struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
	Age   int    `validate:"gte=18"`
}

func TestValidateStruct(t *testing.T) {
	t.Run("valid struct", func(t *testing.T) {
		u := TestUser{
			Name:  "John Doe",
			Email: "john@example.com",
			Age:   25,
		}
		err := ValidateStruct(u)
		assert.NoError(t, err)
	})

	t.Run("invalid struct missing name", func(t *testing.T) {
		u := TestUser{
			Email: "john@example.com",
			Age:   25,
		}
		err := ValidateStruct(u)
		assert.Error(t, err)
	})

	t.Run("invalid struct under age", func(t *testing.T) {
		u := TestUser{
			Name:  "John Doe",
			Email: "john@example.com",
			Age:   17,
		}
		err := ValidateStruct(u)
		assert.Error(t, err)
	})
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email   string
		isValid bool
	}{
		{"test@example.com", true},
		{"user.name+tag@example.co.uk", true},
		{"@example.com", false},
		{"test@", false},
		{"test@example", false},
		{"test", false},
		{"Name <test@example.com>", false}, // Raw email only
	}

	for _, tc := range tests {
		t.Run(tc.email, func(t *testing.T) {
			err := ValidateEmail(tc.email)
			if tc.isValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		password string
		isValid  bool
	}{
		{"P@ssword1", true},
		{"Short1!", false},      // < 8 characters
		{"NoNumbers!", false},   // No numbers
		{"NoSpecial1", false},   // No special character
		{"1234567!@", false},    // No letters
		{"UnicodeL€tt1!", true}, // Unicode letters and special
	}

	for _, tc := range tests {
		t.Run(tc.password, func(t *testing.T) {
			err := ValidatePassword(tc.password)
			if tc.isValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
