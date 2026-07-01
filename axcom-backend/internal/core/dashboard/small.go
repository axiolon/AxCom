// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package dashboard

import (
	"context"
	"time"

	inventoryCore "ecom-engine/internal/core/inventory/features/core"
	"ecom-engine/internal/core/orders"
)

// smallService is the default, zero-dependency dashboard implementation.
// It runs direct queries against the orders and inventory repos with no caching,
// which is appropriate for small stores with low query volume.
type smallService struct {
	orderRepo orders.OrderRepository
	invRepo   inventoryCore.Repository
}

// NewSmallService constructs a SmallDashboardService.
func NewSmallService(orderRepo orders.OrderRepository, invRepo inventoryCore.Repository) Service {
	return &smallService{orderRepo: orderRepo, invRepo: invRepo}
}

func (s *smallService) GetStats(ctx context.Context) (*Stats, error) {
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

	return &Stats{
		Tier:           "small",
		RevenueToday:   revenue,
		OrdersByStatus: counts,
		LowStockSKUs:   lowStockSKUs,
		RecentOrders:   recent,
	}, nil
}
