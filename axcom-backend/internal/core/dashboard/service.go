// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

// Package dashboard implements the admin dashboard service.
// It supports two tiers controlled by config:
//   - "small"  (default): direct queries, no caching — suited for low-volume stores.
//   - "medium": cached aggregations — suited for stores with higher throughput.
//
// TODO: "enterprise" tier with event-driven pre-computation is reserved for a future iteration.
package dashboard

import (
	"context"
	"time"

	"ecom-engine/internal/core/orders"
)

// Service returns aggregated dashboard statistics.
type Service interface {
	GetStats(ctx context.Context) (*Stats, error)
}

// Stats holds aggregated admin dashboard data.
// Medium-tier fields are omitted (nil/empty) for the small tier.
type Stats struct {
	Tier           string                `json:"tier"`
	RevenueToday   float64               `json:"revenue_today"`
	OrdersByStatus map[string]int64      `json:"orders_by_status"`
	LowStockSKUs   []string              `json:"low_stock_skus"`
	RecentOrders   []RecentOrder         `json:"recent_orders"`
	RevenueByDay   []orders.DailyRevenue `json:"revenue_by_day,omitempty"`
	TopProducts    []orders.ProductSales `json:"top_products,omitempty"`
	AOV            *float64              `json:"aov,omitempty"`
}

// RecentOrder is a slim order summary used in the dashboard recent-orders list.
type RecentOrder struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customer_id"`
	Total      float64   `json:"total"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}
