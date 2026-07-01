// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package merge

import (
	"context"
	"errors"

	cartCore "ecom-engine/internal/core/cart"
	"ecom-engine/internal/core/cart/dto"
	"ecom-engine/pkg/logger"
)

// Service defines the business logic for merging guest and account carts.
type Service interface {
	// MergeGuestCartWithAccount merges a guest cart into an authenticated user's account cart.
	MergeGuestCartWithAccount(ctx context.Context, accountCustomerID string, guestCartID string) (*dto.CartResponse, error)
}

type mergeService struct {
	cartService cartCore.Service
	cartRepo    cartCore.Repository
}

// NewMergeService creates and returns an implementation of the Service interface.
func NewMergeService(cartService cartCore.Service, cartRepo cartCore.Repository) Service {
	return &mergeService{
		cartService: cartService,
		cartRepo:    cartRepo,
	}
}

// MergeGuestCartWithAccount merges a guest cart into an authenticated user's account cart.
func (s *mergeService) MergeGuestCartWithAccount(ctx context.Context, accountCustomerID string, guestCartID string) (*dto.CartResponse, error) {
	// Retrieve the guest cart
	guestCart, err := s.cartRepo.GetByCustomerID(ctx, guestCartID)
	if err != nil {
		if errors.Is(err, cartCore.ErrCartNotFound) {
			// No guest cart exists, just return the account cart
			return s.cartService.GetCart(ctx, accountCustomerID)
		}
		return nil, err
	}

	// Merge each guest item via AddItem which validates stock, quantity limits, and unique item caps
	for _, guestItem := range guestCart.Items {
		_, err := s.cartService.AddItem(ctx, accountCustomerID, guestItem)
		if err != nil {
			logger.WarnCtx(ctx, "Skipping guest cart item %s during merge: %v", guestItem.VariantID, err)
		}
	}

	// Delete the guest cart
	_ = s.cartRepo.Delete(ctx, guestCartID)

	// Return the enriched account cart
	return s.cartService.GetCart(ctx, accountCustomerID)
}
