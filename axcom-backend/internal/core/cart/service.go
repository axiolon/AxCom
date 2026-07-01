// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package cart

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"

	"ecom-engine/internal/core/cart/dto"
	"ecom-engine/internal/core/catalog/domain"
	catalogCore "ecom-engine/internal/core/catalog/features/core"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/logger"
	"time"
)

// Service defines the business logic contract for managing shopping carts.
type Service interface {
	// GetCart retrieves the shopping cart for a given customer.
	GetCart(ctx context.Context, customerID string) (*dto.CartResponse, error)

	// AddItem adds an item to the shopping cart or increments quantity if it exists.
	AddItem(ctx context.Context, customerID string, item CartItem) (*dto.CartResponse, error)

	// UpdateItem updates the quantity of a specific variant in the customer's cart.
	UpdateItem(ctx context.Context, customerID string, variantID string, quantity int) (*dto.CartResponse, error)

	// RemoveItem removes a specific variant from the customer's cart.
	RemoveItem(ctx context.Context, customerID string, variantID string) (*dto.CartResponse, error)

	// ClearCart empties the shopping cart for a given customer.
	ClearCart(ctx context.Context, customerID string) error

	// CartCount returns the total number of items in a customer's cart.
	CartCount(ctx context.Context, customerID string) (int, error)

	// CartCountDetailed returns both total items count and distinct items count.
	CartCountDetailed(ctx context.Context, customerID string) (total int, distinct int, err error)
}

type cartService struct {
	repo       Repository
	catalogSvc catalogCore.QueryService
}

// NewCartService creates and returns an implementation of the Service interface.
func NewCartService(repo Repository, catalogSvc catalogCore.QueryService) Service {
	return &cartService{
		repo:       repo,
		catalogSvc: catalogSvc,
	}
}

// GetCart retrieves the shopping cart for a given customer.
func (s *cartService) GetCart(ctx context.Context, customerID string) (*dto.CartResponse, error) {
	c, err := s.getRawCart(ctx, customerID)
	if err != nil {
		return nil, err
	}
	return s.enrichCart(ctx, c)
}

// AddItem adds an item to the shopping cart or increments quantity if it exists.
func (s *cartService) AddItem(ctx context.Context, customerID string, item CartItem) (*dto.CartResponse, error) {
	if item.VariantID == "" {
		logger.ErrorCtx(ctx, "Failed to add item to cart: variant ID is required")
		return nil, apperrors.NewBadRequest("Variant ID is required", ErrVariantIDRequired)
	}
	if item.Quantity <= 0 {
		logger.ErrorCtx(ctx, "Failed to add item to cart: quantity must be > 0 (received %d)", item.Quantity)
		return nil, apperrors.NewBadRequest("Quantity must be greater than zero", ErrInvalidQuantity)
	}

	// 1. Retrieve variant info and stock from Catalog
	p, err := s.catalogSvc.GetProductByVariantID(ctx, item.VariantID)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to retrieve product details for variant %s: %v", item.VariantID, err)
		return nil, apperrors.NewNotFound("Variant not found in catalog", err)
	}

	var targetVar *domain.Variant
	for _, v := range p.Variants {
		if v.ID == item.VariantID {
			vCopy := v
			targetVar = &vCopy
			break
		}
	}
	if targetVar == nil {
		return nil, apperrors.NewNotFound("Variant not found in product", nil)
	}

	c, err := s.getRawCart(ctx, customerID)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to retrieve cart for customer %s: %v", customerID, err)
		return nil, apperrors.NewInternal("failed to retrieve cart", err)
	}

	// Calculate target quantity
	targetQty := item.Quantity
	foundIndex := -1
	for i, existing := range c.Items {
		if existing.VariantID == item.VariantID {
			targetQty += existing.Quantity
			foundIndex = i
			break
		}
	}

	// Ca4-4: Enforce max quantity limit per line item (10 units)
	if targetQty > 10 {
		return nil, apperrors.NewBadRequest("Quantity per item in cart cannot exceed 10", errors.New("quantity limit exceeded"))
	}

	// Validate stock
	if targetQty > targetVar.Stock {
		logger.WarnCtx(ctx, "Insufficient stock for variant %s (requested: %d, available: %d)", item.VariantID, targetQty, targetVar.Stock)
		return nil, apperrors.NewBadRequest(
			fmt.Sprintf("insufficient stock for variant %s (requested: %d, available: %d)", item.VariantID, targetQty, targetVar.Stock),
			errors.New("insufficient stock"),
		)
	}

	// 2. Perform addition/increment
	if foundIndex != -1 {
		c.Items[foundIndex].Quantity = targetQty
	} else {
		// Ca4-3: Enforce max cart unique items limit (50 unique variants)
		if len(c.Items) >= 50 {
			return nil, apperrors.NewBadRequest("Cart cannot contain more than 50 unique items", errors.New("unique items limit exceeded"))
		}
		c.Items = append(c.Items, item)
	}

	c.UpdatedAt = time.Now()
	if err := s.repo.Save(ctx, c); err != nil {
		logger.ErrorCtx(ctx, "Failed to save cart for customer %s: %v", customerID, err)
		return nil, apperrors.NewInternal("failed to save cart changes", err)
	}

	logger.InfoCtx(ctx, "Successfully added item (Variant: %s, Qty: %d) to cart for customer %s", item.VariantID, item.Quantity, customerID)
	return s.enrichCart(ctx, c)
}

// UpdateItem updates the quantity of a specific variant in the customer's cart.
func (s *cartService) UpdateItem(ctx context.Context, customerID string, variantID string, quantity int) (*dto.CartResponse, error) {
	if variantID == "" {
		return nil, apperrors.NewBadRequest("Variant ID is required", ErrVariantIDRequired)
	}
	if quantity <= 0 {
		return nil, apperrors.NewBadRequest("Quantity must be greater than zero", ErrInvalidQuantity)
	}

	// Ca4-4: Enforce max quantity limit per line item (10 units)
	if quantity > 10 {
		return nil, apperrors.NewBadRequest("Quantity per item in cart cannot exceed 10", errors.New("quantity limit exceeded"))
	}

	// 1. Fetch cart and confirm the item exists before touching the catalog.
	c, err := s.getRawCart(ctx, customerID)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to retrieve cart for customer %s: %v", customerID, err)
		return nil, apperrors.NewInternal("failed to retrieve cart", err)
	}

	foundIndex := -1
	for i, existing := range c.Items {
		if existing.VariantID == variantID {
			foundIndex = i
			break
		}
	}
	if foundIndex == -1 {
		return nil, apperrors.NewNotFound("Item not found in cart", errors.New("item not in cart"))
	}

	// 2. Validate stock against catalog only when item is confirmed in the cart.
	p, err := s.catalogSvc.GetProductByVariantID(ctx, variantID)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to retrieve product details for variant %s: %v", variantID, err)
		return nil, apperrors.NewNotFound("Variant not found in catalog", err)
	}

	var targetVar *domain.Variant
	for _, v := range p.Variants {
		if v.ID == variantID {
			vCopy := v
			targetVar = &vCopy
			break
		}
	}
	if targetVar == nil {
		return nil, apperrors.NewNotFound("Variant not found in product", nil)
	}

	// Validate stock
	if quantity > targetVar.Stock {
		logger.WarnCtx(ctx, "Insufficient stock for variant %s (requested: %d, available: %d)", variantID, quantity, targetVar.Stock)
		return nil, apperrors.NewBadRequest(
			fmt.Sprintf("insufficient stock for variant %s (requested: %d, available: %d)", variantID, quantity, targetVar.Stock),
			errors.New("insufficient stock"),
		)
	}

	c.Items[foundIndex].Quantity = quantity

	c.UpdatedAt = time.Now()
	if err := s.repo.Save(ctx, c); err != nil {
		logger.ErrorCtx(ctx, "Failed to save cart for customer %s: %v", customerID, err)
		return nil, apperrors.NewInternal("failed to save cart changes", err)
	}

	logger.InfoCtx(ctx, "Successfully updated item quantity (Variant: %s, Qty: %d) in cart for customer %s", variantID, quantity, customerID)
	return s.enrichCart(ctx, c)
}

// RemoveItem removes a specific variant from the customer's cart.
func (s *cartService) RemoveItem(ctx context.Context, customerID string, variantID string) (*dto.CartResponse, error) {
	if variantID == "" {
		return nil, apperrors.NewBadRequest("Variant ID is required", ErrVariantIDRequired)
	}

	c, err := s.getRawCart(ctx, customerID)
	if err != nil {
		logger.ErrorCtx(ctx, "Failed to retrieve cart for customer %s: %v", customerID, err)
		return nil, apperrors.NewInternal("failed to retrieve cart", err)
	}

	updatedItems := make([]CartItem, 0, len(c.Items))
	for _, existing := range c.Items {
		if existing.VariantID != variantID {
			updatedItems = append(updatedItems, existing)
		}
	}

	// Item was not in the cart — skip the write and return current state.
	if len(updatedItems) == len(c.Items) {
		logger.DebugCtx(ctx, "RemoveItem: variant %s not in cart for customer %s, skipping save", variantID, customerID)
		return s.enrichCart(ctx, c)
	}

	c.Items = updatedItems
	c.UpdatedAt = time.Now()
	if err := s.repo.Save(ctx, c); err != nil {
		logger.ErrorCtx(ctx, "Failed to save cart for customer %s: %v", customerID, err)
		return nil, apperrors.NewInternal("failed to save cart changes", err)
	}

	logger.InfoCtx(ctx, "Successfully removed item (Variant: %s) from cart for customer %s", variantID, customerID)
	return s.enrichCart(ctx, c)
}

// ClearCart empties the shopping cart for a given customer.
func (s *cartService) ClearCart(ctx context.Context, customerID string) error {
	if err := s.repo.Delete(ctx, customerID); err != nil {
		logger.ErrorCtx(ctx, "Failed to clear cart for customer %s: %v", customerID, err)
		return apperrors.NewInternal("failed to empty cart", err)
	}
	logger.InfoCtx(ctx, "Successfully cleared cart for customer %s", customerID)
	return nil
}

func (s *cartService) getRawCart(ctx context.Context, customerID string) (*Cart, error) {
	c, err := s.repo.GetByCustomerID(ctx, customerID)
	if errors.Is(err, ErrCartNotFound) {
		now := time.Now()
		return &Cart{
			CustomerID: customerID,
			Items:      []CartItem{},
			CreatedAt:  now,
			UpdatedAt:  now,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	// Ca4-5: TTL/Expiry mechanism (clear cart items if last updated > 30 days ago)
	if !c.UpdatedAt.IsZero() && time.Since(c.UpdatedAt) > 30*24*time.Hour {
		logger.InfoCtx(ctx, "Cart for customer %s has expired (older than 30 days). Clearing items.", customerID)
		c.Items = []CartItem{}
		c.CreatedAt = time.Now()
		c.UpdatedAt = time.Now()
		if saveErr := s.repo.Save(ctx, c); saveErr != nil {
			logger.ErrorCtx(ctx, "Failed to persist expired cart reset for customer %s: %v", customerID, saveErr)
		}
	}

	return c, nil
}

func (s *cartService) enrichCart(ctx context.Context, c *Cart) (*dto.CartResponse, error) {
	const enrichConcurrency = 5

	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, enrichConcurrency)

	enrichedItems := make([]dto.CartItemResponse, len(c.Items))
	itemExists := make([]bool, len(c.Items))
	var unavailableItems []string

	for i, item := range c.Items {
		wg.Add(1)
		go func(idx int, cItem CartItem) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			p, err := s.catalogSvc.GetProductByVariantID(ctx, cItem.VariantID)
			if err != nil {
				logger.WarnCtx(ctx, "Catalog lookup failed for variant %s in cart: %v", cItem.VariantID, err)
				mu.Lock()
				unavailableItems = append(unavailableItems, cItem.VariantID)
				mu.Unlock()
				return
			}

			var targetVar *domain.Variant
			for _, v := range p.Variants {
				if v.ID == cItem.VariantID {
					vCopy := v
					targetVar = &vCopy
					break
				}
			}

			if targetVar == nil {
				logger.WarnCtx(ctx, "Variant %s not found in product %s variants list", cItem.VariantID, p.ID)
				mu.Lock()
				unavailableItems = append(unavailableItems, cItem.VariantID)
				mu.Unlock()
				return
			}

			discPrice := targetVar.Price
			if p.Discount != nil {
				switch p.Discount.Type {
				case "percentage":
					discPrice = targetVar.Price * (1.0 - p.Discount.Value/100.0)
				case "fixed":
					discPrice = targetVar.Price - p.Discount.Value
					if discPrice < 0 {
						discPrice = 0
					}
				}
			}

			// Round to 2 decimal places
			discPrice = math.Round(discPrice*100) / 100

			var imageURL string
			for _, img := range p.Images {
				if img.IsPrimary {
					imageURL = img.URL
					break
				}
			}
			if imageURL == "" && len(p.Images) > 0 {
				imageURL = p.Images[0].URL
			}

			displayName := targetVar.Name
			if displayName == "" {
				displayName = p.Name
			} else if displayName != p.Name {
				displayName = p.Name + " - " + displayName
			}

			mu.Lock()
			enrichedItems[idx] = dto.CartItemResponse{
				VariantID:       cItem.VariantID,
				Quantity:        cItem.Quantity,
				Name:            displayName,
				SKU:             targetVar.SKU,
				Price:           targetVar.Price,
				DiscountedPrice: discPrice,
				ImageURL:        imageURL,
				Stock:           targetVar.Stock,
				Attributes:      targetVar.Attributes,
			}
			itemExists[idx] = true
			mu.Unlock()
		}(i, item)
	}

	wg.Wait()

	// Filter out unfilled indices
	var items []dto.CartItemResponse
	var totalPrice float64
	var totalDiscountedPrice float64

	for idx, exists := range itemExists {
		if exists {
			items = append(items, enrichedItems[idx])
			totalPrice += enrichedItems[idx].Price * float64(enrichedItems[idx].Quantity)
			totalDiscountedPrice += enrichedItems[idx].DiscountedPrice * float64(enrichedItems[idx].Quantity)
		}
	}

	totalPrice = math.Round(totalPrice*100) / 100
	totalDiscountedPrice = math.Round(totalDiscountedPrice*100) / 100

	return &dto.CartResponse{
		CustomerID:           c.CustomerID,
		Items:                items,
		TotalPrice:           totalPrice,
		TotalDiscountedPrice: totalDiscountedPrice,
		UnavailableItems:     unavailableItems,
	}, nil
}

// CartCount returns the total number of items in a customer's cart.
func (s *cartService) CartCount(ctx context.Context, customerID string) (int, error) {
	c, err := s.getRawCart(ctx, customerID)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, item := range c.Items {
		count += item.Quantity
	}
	return count, nil
}

// CartCountDetailed returns both total items count and distinct items count.
func (s *cartService) CartCountDetailed(ctx context.Context, customerID string) (total int, distinct int, err error) {
	c, err := s.getRawCart(ctx, customerID)
	if err != nil {
		return 0, 0, err
	}

	distinct = len(c.Items)
	count := 0
	for _, item := range c.Items {
		count += item.Quantity
	}
	return count, distinct, nil
}
