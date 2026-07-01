// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package idgen

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Generate creates a cryptographically secure UUIDv7 string with the specified prefix.
func Generate(prefix string) (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("idgen: failed to generate UUIDv7: %w", err)
	}
	return prefix + id.String(), nil
}

// MustGenerate creates a cryptographically secure UUIDv7 string with the specified prefix.
// It panics if UUIDv7 generation fails.
func MustGenerate(prefix string) string {
	id, err := Generate(prefix)
	if err != nil {
		panic(err)
	}
	return id
}

// ToUUID strips the prefix from a prefixed ID and returns the raw uuid.UUID.
// Intended for repository-layer use when DB columns migrate to UUID/BINARY(16).
func ToUUID(prefixedID string) (uuid.UUID, error) {
	idx := strings.Index(prefixedID, "_")
	if idx == -1 {
		// Attempt to parse directly if no prefix is present
		id, err := uuid.Parse(prefixedID)
		if err != nil {
			return uuid.Nil, fmt.Errorf("idgen: failed to parse uuid: %w", err)
		}
		return id, nil
	}

	stripped := prefixedID[idx+1:]
	id, err := uuid.Parse(stripped)
	if err != nil {
		return uuid.Nil, fmt.Errorf("idgen: failed to parse uuid after prefix: %w", err)
	}
	return id, nil
}

// FromUUID re-attaches the prefix to a raw uuid.UUID.
// Intended for repository-layer use when DB columns migrate to UUID/BINARY(16).
func FromUUID(prefix string, id uuid.UUID) string {
	return prefix + id.String()
}

// MustToUUID is like ToUUID but panics on error.
func MustToUUID(prefixedID string) uuid.UUID {
	id, err := ToUUID(prefixedID)
	if err != nil {
		panic(err)
	}
	return id
}
