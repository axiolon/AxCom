// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package cart

import (
	"context"
	"errors"
	"sync"
	"testing"

	"ecom-engine/internal/core/catalog/domain"
	catalogCore "ecom-engine/internal/core/catalog/features/core"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCartRepository implements Repository with RWMutex thread safety
type MockCartRepository struct {
	mu    sync.RWMutex
	carts map[string]*Cart
}

func NewMockCartRepository() *MockCartRepository {
	return &MockCartRepository{
		carts: make(map[string]*Cart),
	}
}

func (m *MockCartRepository) GetByCustomerID(_ context.Context, customerID string) (*Cart, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, exists := m.carts[customerID]
	if !exists {
		return nil, ErrCartNotFound
	}
	copied := &Cart{
		CustomerID: c.CustomerID,
		Items:      make([]CartItem, len(c.Items)),
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
	copy(copied.Items, c.Items)
	return copied, nil
}

func (m *MockCartRepository) Save(_ context.Context, c *Cart) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := &Cart{
		CustomerID: c.CustomerID,
		Items:      make([]CartItem, len(c.Items)),
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

// MockCatalogService implements catalogCore.QueryService
type MockCatalogService struct {
	catalogCore.QueryService
	mu       sync.RWMutex
	products map[string]*domain.Product
}

func NewMockCatalogService() *MockCatalogService {
	return &MockCatalogService{
		products: make(map[string]*domain.Product),
	}
}

func (m *MockCatalogService) GetProductByVariantID(_ context.Context, variantID string) (*domain.Product, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.products {
		for _, v := range p.Variants {
			if v.ID == variantID {
				return p, nil
			}
		}
	}
	return nil, errors.New("variant not found")
}

func (m *MockCatalogService) SeedProduct(p *domain.Product) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.products[p.ID] = p
}

func TestCartService(t *testing.T) {
	t.Parallel()

	setupService := func() (Service, *MockCartRepository, *MockCatalogService) {
		repo := NewMockCartRepository()
		catalogSvc := NewMockCatalogService()

		prod1 := &domain.Product{
			ID:         "prod_1",
			Name:       "Super T-Shirt",
			CategoryID: "cat_1",
			Variants: []domain.Variant{
				{
					ID:    "var_1",
					SKU:   "TSHIRT-RED-L",
					Name:  "Red Large",
					Price: 19.99,
					Stock: 10,
				},
				{
					ID:    "var_2",
					SKU:   "TSHIRT-BLUE-M",
					Name:  "Blue Medium",
					Price: 18.99,
					Stock: 2,
				},
			},
			Discount: &domain.ProductDiscount{
				Type:  "percentage",
				Value: 10.0, // 10% off
			},
			Images: []domain.ProductImage{
				{URL: "https://example.com/primary.jpg", IsPrimary: true},
				{URL: "https://example.com/secondary.jpg", IsPrimary: false},
			},
		}
		catalogSvc.SeedProduct(prod1)

		svc := NewCartService(repo, catalogSvc)
		return svc, repo, catalogSvc
	}

	t.Run("GetCart empty for new customer", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := setupService()
		ctx := context.Background()

		cartResp, err := svc.GetCart(ctx, "cust_1")
		require.NoError(t, err)
		assert.Equal(t, "cust_1", cartResp.CustomerID)
		assert.Empty(t, cartResp.Items)
	})

	t.Run("AddItem successful and applies discount", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := setupService()
		ctx := context.Background()

		cartResp, err := svc.AddItem(ctx, "cust_1", CartItem{VariantID: "var_1", Quantity: 2})
		require.NoError(t, err)
		require.Len(t, cartResp.Items, 1)

		item := cartResp.Items[0]
		assert.Equal(t, "var_1", item.VariantID)
		assert.Equal(t, 2, item.Quantity)
		assert.Equal(t, 19.99, item.Price)
		assert.Equal(t, 17.99, item.DiscountedPrice) // 10% off rounded: 19.99 * 0.9 = 17.991 -> rounded to 17.99
		assert.Equal(t, 10, item.Stock)
		assert.Equal(t, "https://example.com/primary.jpg", item.ImageURL)
		assert.Equal(t, 39.98, cartResp.TotalPrice)
		assert.Equal(t, 35.98, cartResp.TotalDiscountedPrice)
	})

	t.Run("AddItem validation errors", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := setupService()
		ctx := context.Background()

		// Missing Variant ID
		_, err := svc.AddItem(ctx, "cust_1", CartItem{VariantID: "", Quantity: 2})
		assert.Error(t, err)

		// Invalid quantity
		_, err = svc.AddItem(ctx, "cust_1", CartItem{VariantID: "var_1", Quantity: 0})
		assert.Error(t, err)
	})

	t.Run("AddItem insufficient stock", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := setupService()
		ctx := context.Background()

		// Max stock is 10
		_, err := svc.AddItem(ctx, "cust_1", CartItem{VariantID: "var_1", Quantity: 11})
		assert.Error(t, err)
	})

	t.Run("UpdateItem quantity update and validation", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := setupService()
		ctx := context.Background()

		// Seeding the item first
		_, err := svc.AddItem(ctx, "cust_1", CartItem{VariantID: "var_1", Quantity: 1})
		require.NoError(t, err)

		// Successful update
		cartResp, err := svc.UpdateItem(ctx, "cust_1", "var_1", 5)
		require.NoError(t, err)
		require.Len(t, cartResp.Items, 1)
		assert.Equal(t, 5, cartResp.Items[0].Quantity)

		// Update on non-existent item should return error (no silent upsert)
		_, err = svc.UpdateItem(ctx, "cust_1", "var_2", 2)
		assert.Error(t, err)

		// Over-stock update should fail
		_, err = svc.UpdateItem(ctx, "cust_1", "var_1", 15)
		assert.Error(t, err)
	})

	t.Run("RemoveItem variant deletion", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := setupService()
		ctx := context.Background()

		// Add item
		_, err := svc.AddItem(ctx, "cust_1", CartItem{VariantID: "var_1", Quantity: 2})
		require.NoError(t, err)

		// Remove it
		cartResp, err := svc.RemoveItem(ctx, "cust_1", "var_1")
		require.NoError(t, err)
		assert.Empty(t, cartResp.Items)
	})

	t.Run("ClearCart completes successfully", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := setupService()
		ctx := context.Background()

		_, err := svc.AddItem(ctx, "cust_1", CartItem{VariantID: "var_2", Quantity: 1})
		require.NoError(t, err)

		err = svc.ClearCart(ctx, "cust_1")
		require.NoError(t, err)

		cartResp, err := svc.GetCart(ctx, "cust_1")
		require.NoError(t, err)
		assert.Empty(t, cartResp.Items)
	})

	t.Run("CartCount returns correct quantity sum", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := setupService()
		ctx := context.Background()

		_, err := svc.AddItem(ctx, "cust_1", CartItem{VariantID: "var_1", Quantity: 3})
		require.NoError(t, err)

		_, err = svc.AddItem(ctx, "cust_1", CartItem{VariantID: "var_2", Quantity: 2})
		require.NoError(t, err)

		count, err := svc.CartCount(ctx, "cust_1")
		require.NoError(t, err)
		assert.Equal(t, 5, count)
	})

	t.Run("enrichCart discounts and images fallback", func(t *testing.T) {
		t.Parallel()
		repo := NewMockCartRepository()
		catalogSvc := NewMockCatalogService()

		prodFixedDiscount := &domain.Product{
			ID:   "prod_2",
			Name: "Fancy Pants",
			Variants: []domain.Variant{
				{
					ID:    "var_3",
					SKU:   "PANTS-BLUE",
					Name:  "Blue",
					Price: 50.00,
					Stock: 10,
				},
			},
			Discount: &domain.ProductDiscount{
				Type:  "fixed",
				Value: 15.00, // $15 off
			},
			Images: []domain.ProductImage{
				{URL: "https://example.com/fallback.jpg", IsPrimary: false},
			},
		}
		catalogSvc.SeedProduct(prodFixedDiscount)
		svc := NewCartService(repo, catalogSvc)
		ctx := context.Background()

		cartResp, err := svc.AddItem(ctx, "cust_2", CartItem{VariantID: "var_3", Quantity: 1})
		require.NoError(t, err)
		require.Len(t, cartResp.Items, 1)

		item := cartResp.Items[0]
		assert.Equal(t, 50.00, item.Price)
		assert.Equal(t, 35.00, item.DiscountedPrice) // 50 - 15 = 35
		assert.Equal(t, "https://example.com/fallback.jpg", item.ImageURL)
		assert.Equal(t, "Fancy Pants - Blue", item.Name)
	})
}
