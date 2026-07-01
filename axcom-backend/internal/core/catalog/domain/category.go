// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	// ErrInvalidCategoryName is returned when a category name is empty.
	ErrInvalidCategoryName = errors.New("category name is required")

	// ErrInvalidCategorySlug is returned when a category slug is empty.
	ErrInvalidCategorySlug = errors.New("category slug is required")

	// ErrInvalidCategory is returned when a category structure is invalid.
	ErrInvalidCategory = errors.New("invalid category name or slug")
)

// Category represents a grouping for products.
type Category struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	ParentID  *string   `json:"parent_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ValidateCategory checks if the category is valid.
func ValidateCategory(c Category) error {
	if strings.TrimSpace(c.Name) == "" {
		return ErrInvalidCategoryName
	}
	if strings.TrimSpace(c.Slug) == "" {
		return ErrInvalidCategorySlug
	}
	return nil
}

var nonAlphaNumRegex = regexp.MustCompile(`[^a-z0-9\-]+`)
var multiHyphenRegex = regexp.MustCompile(`-+`)

// GenerateSlug creates a URL-friendly slug from a category name.
func GenerateSlug(name string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	normalized, _, _ := transform.String(t, name)

	s := strings.ToLower(normalized)
	// Replace spaces and underscores with hyphens
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	// Remove non-alphanumeric and non-hyphen characters
	s = nonAlphaNumRegex.ReplaceAllString(s, "")
	// Replace multiple consecutive hyphens with a single one
	s = multiHyphenRegex.ReplaceAllString(s, "-")
	// Trim hyphens from ends
	return strings.Trim(s, "-")
}
