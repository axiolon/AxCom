// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package merge

import (
	"context"
	"sync"
	"testing"

	cartCore "ecom-engine/internal/core/cart"
	"ecom-engine/internal/core/cart/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Re-define thread-safe mocks for this package since they are localized to cart package

type MockCartRepository struct {
	mu    sync.RWMutex
	carts map[string]*cartCore.Cart
}

func NewMockCartRepository() *MockCartRepository {
	return &MockCartRepository{
		carts: make(map[string]*cartCore.Cart),
	}
}

func (m *MockCartRepository) GetByCustomerID(_ context.Context, customerID string) (*cartCore.Cart, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, exists := m.carts[customerID]
	if !exists {
		return nil, cartCore.ErrCartNotFound
	}
	copied := &cartCore.Cart{
		CustomerID: c.CustomerID,
		Items:      make([]cartCore.CartItem, len(c.Items)),
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
	copy(copied.Items, c.Items)
	return copied, nil
}

func (m *MockCartRepository) Save(_ context.Context, c *cartCore.Cart) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := &cartCore.Cart{
		CustomerID: c.CustomerID,
		Items:      make([]cartCore.CartItem, len(c.Items)),
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
	copy(copied.Items, c.Items)
	m.carts[c.CustomerID] = copied
	return nil
}

func (m *MockCartRepository) Delete(_ context.Context, customerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.carts, customerID)
	return nil
}

func (m *MockCartRepository) Exists(_ context.Context, customerID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.carts[customerID]
	return exists, nil
}

type MockCartService struct {
	cartCore.Service
	repo        cartCore.Repository
	GetCartFunc func(ctx context.Context, customerID string) (*dto.CartResponse, error)
}

func (m *MockCartService) GetCart(ctx context.Context, customerID string) (*dto.CartResponse, error) {
	if m.GetCartFunc != nil {
		return m.GetCartFunc(ctx, customerID)
	}
	return nil, nil
}

func (m *MockCartService) AddItem(ctx context.Context, customerID string, item cartCore.CartItem) (*dto.CartResponse, error) {
	c, err := m.repo.GetByCustomerID(ctx, customerID)
	if err != nil {
		c = &cartCore.Cart{
			CustomerID: customerID,
			Items:      []cartCore.CartItem{},
		}
	}
	found := false
	for i, existing := range c.Items {
		if existing.VariantID == item.VariantID {
			c.Items[i].Quantity += item.Quantity
			found = true
			break
		}
	}
	if !found {
		c.Items = append(c.Items, item)
	}
	err = m.repo.Save(ctx, c)
	if err != nil {
		return nil, err
	}
	return &dto.CartResponse{CustomerID: customerID}, nil
}

func TestMergeService(t *testing.T) {
	t.Parallel()

	t.Run("MergeGuestCartWithAccount - guest cart not found", func(t *testing.T) {
		t.Parallel()
		repo := NewMockCartRepository()

		// Setup mock cart service that returns an empty cart for account
		cartSvc := &MockCartService{
			repo: repo,
			GetCartFunc: func(_ context.Context, customerID string) (*dto.CartResponse, error) {
				return &dto.CartResponse{
					CustomerID: customerID,
					Items:      []dto.CartItemResponse{},
				}, nil
			},
		}

		svc := NewMergeService(cartSvc, repo)
		ctx := context.Background()

		resp, err := svc.MergeGuestCartWithAccount(ctx, "account_123", "nonexistent_guest")
		require.NoError(t, err)
		assert.Equal(t, "account_123", resp.CustomerID)
		assert.Empty(t, resp.Items)
	})

	t.Run("MergeGuestCartWithAccount - success merge with new account cart", func(t *testing.T) {
		t.Parallel()
		repo := NewMockCartRepository()

		// Save a guest cart with items
		guestCart := &cartCore.Cart{
			CustomerID: "guest_123",
			Items: []cartCore.CartItem{
				{VariantID: "var_1", Quantity: 2},
				{VariantID: "var_2", Quantity: 3},
			},
		}
		repo.carts["guest_123"] = guestCart

		cartSvc := &MockCartService{
			repo: repo,
			GetCartFunc: func(_ context.Context, customerID string) (*dto.CartResponse, error) {
				// We expect guest items to have merged into account cart in database
				c, _ := repo.GetByCustomerID(context.Background(), customerID)
				itemsResp := make([]dto.CartItemResponse, len(c.Items))
				for i, it := range c.Items {
					itemsResp[i] = dto.CartItemResponse{
						VariantID: it.VariantID,
						Quantity:  it.Quantity,
					}
				}
				return &dto.CartResponse{
					CustomerID: customerID,
					Items:      itemsResp,
				}, nil
			},
		}

		svc := NewMergeService(cartSvc, repo)
		ctx := context.Background()

		resp, err := svc.MergeGuestCartWithAccount(ctx, "account_123", "guest_123")
		require.NoError(t, err)
		assert.Equal(t, "account_123", resp.CustomerID)
		require.Len(t, resp.Items, 2)

		// Assertions on the merged repository state
		savedAccountCart, err := repo.GetByCustomerID(ctx, "account_123")
		require.NoError(t, err)
		assert.Len(t, savedAccountCart.Items, 2)
		assert.Equal(t, "var_1", savedAccountCart.Items[0].VariantID)
		assert.Equal(t, 2, savedAccountCart.Items[0].Quantity)

		// Assert guest cart was deleted
		_, err = repo.GetByCustomerID(ctx, "guest_123")
		assert.Error(t, err)
	})

	t.Run("MergeGuestCartWithAccount - merge overlapping items", func(t *testing.T) {
		t.Parallel()
		repo := NewMockCartRepository()

		// Guest cart
		repo.carts["guest_123"] = &cartCore.Cart{
			CustomerID: "guest_123",
			Items: []cartCore.CartItem{
				{VariantID: "var_1", Quantity: 2},
				{VariantID: "var_2", Quantity: 1},
			},
		}

		// Pre-existing Account cart
		repo.carts["account_123"] = &cartCore.Cart{
			CustomerID: "account_123",
			Items: []cartCore.CartItem{
				{VariantID: "var_1", Quantity: 5},
			},
		}

		cartSvc := &MockCartService{
			repo: repo,
			GetCartFunc: func(_ context.Context, customerID string) (*dto.CartResponse, error) {
				return &dto.CartResponse{CustomerID: customerID}, nil
			},
		}

		svc := NewMergeService(cartSvc, repo)
		ctx := context.Background()

		_, err := svc.MergeGuestCartWithAccount(ctx, "account_123", "guest_123")
		require.NoError(t, err)

		savedAccount, err := repo.GetByCustomerID(ctx, "account_123")
		require.NoError(t, err)

		// var_1: 5 + 2 = 7; var_2: 1
		require.Len(t, savedAccount.Items, 2)
		assert.Equal(t, "var_1", savedAccount.Items[0].VariantID)
		assert.Equal(t, 7, savedAccount.Items[0].Quantity)
		assert.Equal(t, "var_2", savedAccount.Items[1].VariantID)
		assert.Equal(t, 1, savedAccount.Items[1].Quantity)
	})
}
