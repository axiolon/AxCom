// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package cart

// CacheKeys provides centralized cache key generation for Cart module.
// This prevents cache key collisions and makes invalidation easier.
type CacheKeys struct{}

// CartByUserID generates a cache key for a user's cart.
// Format: "cart:user:{userID}"
func (ck *CacheKeys) CartByUserID(userID string) string {
	return "cart:user:" + userID
}

// CartMergeState generates a cache key for a merge operation's temporary state.
// Format: "cart:merge:{sessionID}"
func (ck *CacheKeys) CartMergeState(sessionID string) string {
	return "cart:merge:" + sessionID
}

// NewCacheKeys creates a new cache keys helper.
func NewCacheKeys() *CacheKeys {
	return &CacheKeys{}
}
