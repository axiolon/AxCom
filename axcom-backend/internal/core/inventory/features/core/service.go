// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ecom-engine/internal/core/inventory/domain"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/idgen"
	"ecom-engine/pkg/logger"
)

// ConfigureStockSettings holds the configuration fields that can be updated
// or initialized for a variant's stock item at a specific location.
type ConfigureStockSettings struct {
	VariantID         string // Unique identifier of the product variant
	LocationID        string // Inventory location (e.g., "default", "warehouse-1")
	Quantity          *int   // Optional initial stock quantity
	LowStockThreshold *int   // Optional quantity threshold for low stock alert
	AllowBackorders   *bool  // Optional setting to permit backordering
	BackorderLimit    *int   // Optional limit on maximum backordered units allowed
}

// Service defines the business operations interface for core inventory management.
type Service interface {
	// UpdateStock updates the quantity of a specific variant at a location.
	UpdateStock(ctx context.Context, variantID string, locationID string, qty int) error
	// CheckStock retrieves the current quantity available for a variant at a location.
	CheckStock(ctx context.Context, variantID string, locationID string) (int, error)
	// DeleteStock deletes the stock mapping details for a variant and location.
	DeleteStock(ctx context.Context, variantID string, locationID string) error
	// ListStock lists stock items according to filter parameters.
	ListStock(ctx context.Context, filter ListStockFilter) ([]*domain.StockItem, error)
	// ListAlerts retrieves all triggered low stock alerts.
	ListAlerts(ctx context.Context, limit, offset int) ([]*domain.Alert, error)
	// ConfigureStock configures alerts, backorders, and thresholds for a variant's stock.
	ConfigureStock(ctx context.Context, settings ConfigureStockSettings) error
}

// inventoryService implements the Service interface.
type inventoryService struct {
	repo       Repository             // Storage repository for stock records
	dispatcher domain.AlertDispatcher // Dispatcher for outgoing stock notifications
}

// NewService initializes a new instance of the core inventory Service.
func NewService(repo Repository, dispatcher domain.AlertDispatcher) Service {
	return &inventoryService{
		repo:       repo,
		dispatcher: dispatcher,
	}
}

// UpdateStock changes the stock level for a variant at a location. It also
// checks and fires an alert if the stock drops below the low stock threshold.
func (s *inventoryService) UpdateStock(ctx context.Context, variantID string, locationID string, qty int) error {
	// Default to "default" location if empty
	if locationID == "" {
		locationID = "default"
	}
	logger.InfoCtx(ctx, "Updating stock level for variant %s at location %s to %d", variantID, locationID, qty)

	// Fetch existing stock record or initialize a default template if missing
	stock, err := s.repo.GetStock(ctx, variantID, locationID)
	if err != nil {
		stock = &domain.StockItem{
			VariantID:         variantID,
			LocationID:        locationID,
			Quantity:          0,
			LowStockThreshold: domain.DefaultLowStockThreshold,
			AllowBackorders:   false,
			BackorderLimit:    0,
		}
	}

	// Update the stock item entity through domain validation rules
	if err := stock.Update(qty); err != nil {
		logger.ErrorCtx(ctx, "Invalid stock update for variant %s at location %s to %d: %v", variantID, locationID, qty, err)
		if errors.Is(err, domain.ErrInvalidQuantity) {
			return apperrors.NewBadRequest("stock quantity cannot be negative", domain.ErrInvalidQuantity)
		}
		return apperrors.NewBadRequest("stock update failed validation", err)
	}

	// Persist the changes
	if err := s.repo.SaveStock(ctx, stock); err != nil {
		logger.ErrorCtx(ctx, "Failed to save stock update for variant %s at location %s: %v", variantID, locationID, err)
		return apperrors.NewInternal("failed to save stock levels", err)
	}

	// Trigger alerts if the new stock quantity is below low-stock thresholds
	s.checkAndDispatchAlert(ctx, stock)

	logger.InfoCtx(ctx, "Successfully updated stock for variant %s at location %s to %d", variantID, locationID, qty)
	return nil
}

// CheckStock returns the available quantity of a variant at a location, returning 0 if not configured.
func (s *inventoryService) CheckStock(ctx context.Context, variantID string, locationID string) (int, error) {
	if locationID == "" {
		locationID = "default"
	}
	logger.InfoCtx(ctx, "Checking stock level for variant %s at location %s", variantID, locationID)

	stock, err := s.repo.GetStock(ctx, variantID, locationID)
	if err != nil {
		logger.InfoCtx(ctx, "Stock record not found for variant %s at location %s, returning 0", variantID, locationID)
		return 0, nil
	}

	return stock.Quantity, nil
}

// DeleteStock removes the stock record mapping for the specified variant and location.
func (s *inventoryService) DeleteStock(ctx context.Context, variantID string, locationID string) error {
	if locationID == "" {
		locationID = "default"
	}
	logger.InfoCtx(ctx, "Deleting stock for variant %s at location %s", variantID, locationID)
	if err := s.repo.DeleteStock(ctx, variantID, locationID); err != nil {
		logger.ErrorCtx(ctx, "Failed to delete stock for variant %s at location %s: %v", variantID, locationID, err)
		return apperrors.NewInternal("failed to delete stock", err)
	}
	return nil
}

// ListStock retrieves stock items filtered by properties like location ID or status.
func (s *inventoryService) ListStock(ctx context.Context, filter ListStockFilter) ([]*domain.StockItem, error) {
	logger.InfoCtx(ctx, "Listing stock items with filters: %+v", filter)
	stocks, err := s.repo.ListStock(ctx, filter)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to list stock: %v", err)
		return nil, apperrors.NewInternal("failed to retrieve stock list", err)
	}
	return stocks, nil
}

// ListAlerts retrieves all active triggered low-stock alerts.
func (s *inventoryService) ListAlerts(ctx context.Context, limit, offset int) ([]*domain.Alert, error) {
	logger.InfoCtx(ctx, "Listing all stock alerts with limit: %d, offset: %d", limit, offset)
	alerts, err := s.repo.ListAlerts(ctx, limit, offset)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to list alerts: %v", err)
		return nil, apperrors.NewInternal("failed to retrieve alerts", err)
	}
	return alerts, nil
}

// ConfigureStock updates low-stock thresholds, backorders, and initial quantities for a stock item.
func (s *inventoryService) ConfigureStock(ctx context.Context, settings ConfigureStockSettings) error {
	if settings.LocationID == "" {
		settings.LocationID = "default"
	}
	logger.InfoCtx(ctx, "Configuring stock properties for variant %s at location %s", settings.VariantID, settings.LocationID)

	// Fetch existing stock record or initialize a default template if missing
	stock, err := s.repo.GetStock(ctx, settings.VariantID, settings.LocationID)
	if err != nil {
		stock = &domain.StockItem{
			VariantID:         settings.VariantID,
			LocationID:        settings.LocationID,
			Quantity:          0,
			LowStockThreshold: domain.DefaultLowStockThreshold,
			AllowBackorders:   false,
			BackorderLimit:    0,
		}
	}

	// Apply configuration updates
	if settings.Quantity != nil {
		if err := stock.Update(*settings.Quantity); err != nil {
			return err
		}
	}
	if settings.LowStockThreshold != nil {
		stock.LowStockThreshold = *settings.LowStockThreshold
	}
	if settings.AllowBackorders != nil {
		stock.AllowBackorders = *settings.AllowBackorders
	}
	if settings.BackorderLimit != nil {
		stock.BackorderLimit = *settings.BackorderLimit
	}

	// Persist the changes
	if err := s.repo.SaveStock(ctx, stock); err != nil {
		logger.ErrorCtx(ctx, "Failed to save configured stock variant %s: %v", settings.VariantID, err)
		return apperrors.NewInternal("failed to save stock settings", err)
	}

	// Check and fire alerts if configuration settings caused stock levels to drop below threshold
	s.checkAndDispatchAlert(ctx, stock)

	return nil
}

// checkAndDispatchAlert verifies if stock is low and dispatches a new alert through the dispatcher.
func (s *inventoryService) checkAndDispatchAlert(ctx context.Context, stock *domain.StockItem) {
	if stock.LowStockThreshold <= 0 {
		stock.LowStockThreshold = domain.DefaultLowStockThreshold
	}

	if stock.IsLowStock() {
		alertID, err := idgen.Generate("alt_")
		if err != nil {
			logger.ErrorCtx(ctx, "Failed to generate alert ID for variant %s: %v", stock.VariantID, err)
			return
		}
		alert := domain.Alert{
			ID:        alertID,
			Type:      "LOW_STOCK",
			Message:   fmt.Sprintf("Stock for variant %s has fallen below threshold. Current stock: %d", stock.VariantID, stock.Quantity),
			VariantID: stock.VariantID,
			CreatedAt: time.Now(),
			IsRead:    false,
		}
		if err := s.dispatcher.Dispatch(ctx, alert); err != nil {
			logger.ErrorCtx(ctx, "Failed to dispatch low stock alert for variant %s: %v", stock.VariantID, err)
		}
	}
}
