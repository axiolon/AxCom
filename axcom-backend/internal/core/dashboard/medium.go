// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	inventoryCore "ecom-engine/internal/core/inventory/features/core"
	"ecom-engine/internal/core/orders"
	"ecom-engine/internal/infra/cache"
)

const cacheKey = "dashboard:stats"

// mediumService extends small-tier stats with 30-day revenue charts,
// top products, and AOV. Results are cached for CacheTTL to avoid
// repeated aggregations on busy stores.
type mediumService struct {
	orderRepo orders.OrderRepository
	invRepo   inventoryCore.Repository
	cache     cache.Cache
	ttl       time.Duration
}

// NewMediumService constructs a MediumDashboardService.
func NewMediumService(orderRepo orders.OrderRepository, invRepo inventoryCore.Repository, c cache.Cache, ttl time.Duration) Service {
	return &mediumService{orderRepo: orderRepo, invRepo: invRepo, cache: c, ttl: ttl}
}

func (s *mediumService) GetStats(ctx context.Context) (*Stats, error) {
	// Serve from cache when available.
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var stats Stats
		if json.Unmarshal([]byte(cached), &stats) == nil {
			return &stats, nil
		}
	} else if !errors.Is(err, cache.ErrCacheMiss) {
		// Backend error — fall through and compute live.
		_ = err
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	revenue, err := s.orderRepo.SumRevenue(ctx, today)
	if err != nil {
		return nil, err
	}

	counts, err := s.orderRepo.CountByStatus(ctx)
	if err != nil {
		return nil, err
	}

	recentOrders, err := s.orderRepo.ListAll(ctx, 10, 0)
	if err != nil {
		return nil, err
	}

	alerts, err := s.invRepo.ListAlerts(ctx, 20, 0)
	if err != nil {
		return nil, err
	}

	byDay, err := s.orderRepo.RevenueByDay(ctx, 30)
	if err != nil {
		return nil, err
	}

	topProds, err := s.orderRepo.TopProducts(ctx, 10)
	if err != nil {
		return nil, err
	}

	lowStockSKUs := make([]string, 0, len(alerts))
	for _, a := range alerts {
		if !a.IsRead {
			lowStockSKUs = append(lowStockSKUs, a.VariantID)
		}
	}

	recent := make([]RecentOrder, len(recentOrders))
	for i, o := range recentOrders {
		recent[i] = RecentOrder{
			ID:         o.ID,
			CustomerID: o.CustomerID,
			Total:      o.Total,
			Status:     string(o.Status),
			CreatedAt:  o.CreatedAt,
		}
	}

	// AOV = 30-day total revenue / all-time order count (approximate but useful).
	var aov *float64
	totalRevenue30 := 0.0
	for _, d := range byDay {
		totalRevenue30 += d.Revenue
	}
	totalOrders := int64(0)
	for _, c := range counts {
		totalOrders += c
	}
	if totalOrders > 0 {
		v := totalRevenue30 / float64(totalOrders)
		aov = &v
	}

	stats := &Stats{
		Tier:           "medium",
		RevenueToday:   revenue,
		OrdersByStatus: counts,
		LowStockSKUs:   lowStockSKUs,
		RecentOrders:   recent,
		RevenueByDay:   byDay,
		TopProducts:    topProds,
		AOV:            aov,
	}

	// Best-effort cache write — never fail the request on cache errors.
	if data, err := json.Marshal(stats); err == nil {
		_ = s.cache.Set(ctx, cacheKey, string(data), s.ttl)
	}

	return stats, nil
}
