// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package orders contains the core service, domain models, validation, and state machine transitions for managing orders in the system.
package orders

import (
	"context"
	"time"
)

// DailyRevenue holds the total revenue for a single calendar day.
type DailyRevenue struct {
	Date    string  `json:"date"` // "YYYY-MM-DD"
	Revenue float64 `json:"revenue"`
}

// ProductSales holds aggregated sales for a single product variant.
type ProductSales struct {
	VariantID string `json:"variant_id"`
	TotalSold int64  `json:"total_sold"`
}

// OrderRepository defines the interface for order persistence operations.
type OrderRepository interface {
	// Create persists a new order in the repository.
	Create(ctx context.Context, o *Order) error

	// GetByID retrieves an order by its unique identifier.
	GetByID(ctx context.Context, id string) (*Order, error)

	// Update updates the mutable fields of an existing order.
	Update(ctx context.Context, o *Order) error

	// ListByCustomerID retrieves a paginated slice of orders belonging to a specific customer.
	ListByCustomerID(ctx context.Context, customerID string, limit, offset int) ([]Order, error)

	// ListAll retrieves a paginated slice of all orders in the system.
	ListAll(ctx context.Context, limit, offset int) ([]Order, error)

	// CountByStatus returns the total number of orders grouped by status.
	CountByStatus(ctx context.Context) (map[string]int64, error)

	// SumRevenue returns the total revenue from orders created at or after since.
	// Pass a zero time.Time to sum all orders regardless of date.
	SumRevenue(ctx context.Context, since time.Time) (float64, error)

	// RevenueByDay returns daily revenue totals for the last days calendar days.
	// Results are ordered oldest-first.
	RevenueByDay(ctx context.Context, days int) ([]DailyRevenue, error)

	// TopProducts returns the top n product variants by total units sold.
	TopProducts(ctx context.Context, n int) ([]ProductSales, error)
}
