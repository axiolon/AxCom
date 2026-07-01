// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package ctxkeys

// ContextKey defines a custom string type to avoid collision of context keys.
type ContextKey string

const (
	// UserIDKey is the context key for retrieving the authenticated user's ID.
	UserIDKey ContextKey = "user_id"

	// UserRoleKey is the context key for retrieving the authenticated user's role.
	UserRoleKey ContextKey = "user_role"

	// CorrelationIDKey is the context key for retrieving request/event correlation IDs.
	CorrelationIDKey ContextKey = "correlation_id"
)
