// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package guest defines domain models and validation logic for guest customer data.
package guest

import (
	"errors"
	"net/mail"
	"regexp"
)

// GuestCustomerInfo holds contact details for unauthenticated guest checkouts.
type GuestCustomerInfo struct { //nolint:revive // Name is intentionally explicit for the public API.
	Name          string `json:"name"`
	Email         string `json:"email"`
	ContactNumber string `json:"contact_number"`
}

var (
	// ErrGuestInfoRequired is returned when guest info is missing for guest checkout.
	ErrGuestInfoRequired = errors.New("guest info is required for guest checkout")

	// ErrGuestNameRequired is returned when guest name is missing.
	ErrGuestNameRequired = errors.New("guest name is required")

	// ErrGuestEmailRequired is returned when guest email is missing.
	ErrGuestEmailRequired = errors.New("guest email is required")

	// ErrGuestEmailInvalid is returned when guest email is invalid.
	ErrGuestEmailInvalid = errors.New("guest email is invalid")

	// ErrGuestContactRequired is returned when guest contact number is missing.
	ErrGuestContactRequired = errors.New("guest contact number is required")

	// ErrGuestContactInvalid is returned when guest contact number is invalid.
	ErrGuestContactInvalid = errors.New("guest contact number is invalid")
)

var phoneRegex = regexp.MustCompile(`^\+?[0-9\s\-]{7,15}$`)

// ValidateGuestInfo validates that guest customer info is non-nil and all fields are provided.
func ValidateGuestInfo(g *GuestCustomerInfo) error {
	if g == nil {
		return ErrGuestInfoRequired
	}
	if g.Name == "" {
		return ErrGuestNameRequired
	}
	if g.Email == "" {
		return ErrGuestEmailRequired
	}
	if _, err := mail.ParseAddress(g.Email); err != nil {
		return ErrGuestEmailInvalid
	}
	if g.ContactNumber == "" {
		return ErrGuestContactRequired
	}
	if !phoneRegex.MatchString(g.ContactNumber) {
		return ErrGuestContactInvalid
	}
	return nil
}
