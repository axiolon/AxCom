// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"ecom-engine/internal/core/catalog/domain"
	"ecom-engine/internal/events"
	"ecom-engine/internal/infra/cache"
	apperrors "ecom-engine/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockCacheManager struct{}

var _ cache.Manager = (*MockCacheManager)(nil)

func (m *MockCacheManager) GetOrFetch(_ context.Context, _ string, target interface{}, _ time.Duration, fetchFn func() (interface{}, error)) error {
	data, err := fetchFn()
	if err != nil {
		return err
	}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}

func (m *MockCacheManager) Invalidate(_ context.Context, _ string) error {
	return nil
}

func (m *MockCacheManager) Close() error {
	return nil
}

// MockRepository implements core.Repository
type MockRepository struct {
	mu            sync.RWMutex
	products      map[string]*domain.Product
	categories    map[string]*domain.Category
	createProdErr error
	getProdErr    error
	listProdErr   error
	updateProdErr error
	deleteProdErr error
	createCatErr  error
	getCatErr     error
	listCatErr    error
	updateCatErr  error
	deleteCatErr  error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		products:   make(map[string]*domain.Product),
		categories: make(map[string]*domain.Category),
	}
}

func (m *MockRepository) CreateProduct(_ context.Context, p *domain.Product) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createProdErr != nil {
		return m.createProdErr
	}
	m.products[p.ID] = p
	return nil
}

func (m *MockRepository) GetProductByID(_ context.Context, id string) (*domain.Product, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getProdErr != nil {
		return nil, m.getProdErr
	}
	p, exists := m.products[id]
	if !exists {
		return nil, ErrProductNotFound
	}
	return p, nil
}

func (m *MockRepository) ListProducts(_ context.Context, filter *ProductFilter) ([]domain.Product, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.listProdErr != nil {
		return nil, m.listProdErr
	}
	var list []domain.Product
	for _, p := range m.products {
		list = append(list, *p)
	}
	if filter != nil {
		if filter.Limit > 0 {
			end := filter.Offset + filter.Limit
			if end > int64(len(list)) {
				end = int64(len(list))
			}
			if filter.Offset < int64(len(list)) {
				list = list[filter.Offset:end]
			} else {
				list = []domain.Product{}
			}
		}
	}
	return list, nil
}

func (m *MockRepository) UpdateProduct(_ context.Context, p *domain.Product) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateProdErr != nil {
		return m.updateProdErr
	}
	existing, ok := m.products[p.ID]
	if !ok {
		return ErrProductNotFound
	}
	if existing.Version != p.Version {
		return ErrVersionConflict
	}
	p.Version++
	m.products[p.ID] = p
	return nil
}

func (m *MockRepository) DeleteProduct(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deleteProdErr != nil {
		return m.deleteProdErr
	}
	if _, exists := m.products[id]; !exists {
		return ErrProductNotFound
	}
	delete(m.products, id)
	return nil
}

func (m *MockRepository) GetProductByVariantID(_ context.Context, variantID string) (*domain.Product, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getProdErr != nil {
		return nil, m.getProdErr
	}
	for _, p := range m.products {
		for _, v := range p.Variants {
			if v.ID == variantID {
				return p, nil
			}
		}
	}
	return nil, ErrProductNotFound
}

func (m *MockRepository) UpdateVariantStock(_ context.Context, variantID string, stock int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateProdErr != nil {
		return m.updateProdErr
	}
	for _, p := range m.products {
		for i, v := range p.Variants {
			if v.ID == variantID {
				p.Variants[i].Stock = stock
				return nil
			}
		}
	}
	return ErrProductNotFound
}

func (m *MockRepository) CreateCategory(_ context.Context, c *domain.Category) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createCatErr != nil {
		return m.createCatErr
	}
	m.categories[c.ID] = c
	return nil
}

func (m *MockRepository) GetCategoryByID(_ context.Context, id string) (*domain.Category, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getCatErr != nil {
		return nil, m.getCatErr
	}
	c, exists := m.categories[id]
	if !exists {
		return nil, ErrCategoryNotFound
	}
	return c, nil
}

func (m *MockRepository) ListCategories(_ context.Context) ([]domain.Category, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.listCatErr != nil {
		return nil, m.listCatErr
	}
	var list []domain.Category
	for _, c := range m.categories {
		list = append(list, *c)
	}
	return list, nil
}

func (m *MockRepository) UpdateCategory(_ context.Context, c *domain.Category) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateCatErr != nil {
		return m.updateCatErr
	}
	m.categories[c.ID] = c
	return nil
}

func (m *MockRepository) DeleteCategory(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deleteCatErr != nil {
		return m.deleteCatErr
	}
	if _, exists := m.categories[id]; !exists {
		return ErrCategoryNotFound
	}
	delete(m.categories, id)
	return nil
}

// MockEventBus implements events.EventBus with synchronous triggers for fast testing
type MockEventBus struct {
	mu          sync.RWMutex
	published   []events.Event
	subscribers map[string][]events.EventHandler
}

func NewMockEventBus() *MockEventBus {
	return &MockEventBus{
		subscribers: make(map[string][]events.EventHandler),
	}
}

func (m *MockEventBus) Subscribe(topic string, handler events.EventHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribers[topic] = append(m.subscribers[topic], handler)
}

func (m *MockEventBus) Publish(event events.Event) {
	m.mu.Lock()
	m.published = append(m.published, event)
	m.mu.Unlock()

	m.mu.RLock()
	handlers, exists := m.subscribers[event.Topic]
	m.mu.RUnlock()
	if exists {
		for _, handler := range handlers {
			_ = handler(event) // Execute synchronously in test
		}
	}
}

func (m *MockEventBus) Close() error {
	return nil
}

func (m *MockEventBus) GetPublished() []events.Event {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.published
}

func TestCatalogService_AddProduct(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*MockRepository, CommandService) {
		repo := NewMockRepository()
		svc := NewCatalogCommandService(repo, &MockCacheManager{})
		return repo, svc
	}

	t.Run("successful product creation", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		// Create category first
		cat := &domain.Category{ID: "cat_1", Name: "Electronics"}
		repo.categories[cat.ID] = cat

		p := &domain.Product{
			Name:        "Smartphone",
			Description: "Latest model",
			CategoryID:  "cat_1",
			Variants: []domain.Variant{
				{SKU: "SM-PHN-01", Name: "Standard", Price: 699.99},
			},
		}

		err := svc.AddProduct(context.Background(), p)
		require.NoError(t, err)
		assert.NotEmpty(t, p.ID)
		assert.NotEmpty(t, p.Variants[0].ID)

		stored, err := repo.GetProductByID(context.Background(), p.ID)
		require.NoError(t, err)
		assert.Equal(t, "Smartphone", stored.Name)
	})

	t.Run("fails - missing category ID", func(t *testing.T) {
		t.Parallel()
		_, svc := setup(t)

		p := &domain.Product{
			Name: "Smartphone",
			Variants: []domain.Variant{
				{SKU: "SM-PHN-01", Name: "Standard", Price: 699.99},
			},
		}

		err := svc.AddProduct(context.Background(), p)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.Code)
	})

	t.Run("fails - non-existent category", func(t *testing.T) {
		t.Parallel()
		_, svc := setup(t)

		p := &domain.Product{
			Name:       "Smartphone",
			CategoryID: "cat_nonexistent",
			Variants: []domain.Variant{
				{SKU: "SM-PHN-01", Name: "Standard", Price: 699.99},
			},
		}

		err := svc.AddProduct(context.Background(), p)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.Code)
	})
}

func TestCatalogService_UpdateProduct(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*MockRepository, CommandService) {
		repo := NewMockRepository()
		svc := NewCatalogCommandService(repo, &MockCacheManager{})
		return repo, svc
	}

	t.Run("successful product update", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		cat := &domain.Category{ID: "cat_1", Name: "Electronics"}
		repo.categories[cat.ID] = cat

		p := &domain.Product{
			ID:         "prod_1",
			Name:       "Old Smartphone",
			CategoryID: "cat_1",
			Variants: []domain.Variant{
				{ID: "var_1", SKU: "SM-PHN-01", Name: "Standard", Price: 699.99},
			},
		}
		repo.products[p.ID] = p

		pUpdate := &domain.Product{
			ID:         "prod_1",
			Name:       "New Smartphone",
			CategoryID: "cat_1",
			Variants: []domain.Variant{
				{ID: "var_1", SKU: "SM-PHN-01", Name: "Standard", Price: 749.99},
			},
		}

		err := svc.UpdateProduct(context.Background(), pUpdate)
		require.NoError(t, err)

		stored, _ := repo.GetProductByID(context.Background(), "prod_1")
		assert.Equal(t, "New Smartphone", stored.Name)
	})

	t.Run("fails - missing product ID", func(t *testing.T) {
		t.Parallel()
		_, svc := setup(t)

		pUpdate := &domain.Product{
			Name: "New Smartphone",
		}

		err := svc.UpdateProduct(context.Background(), pUpdate)
		assert.Error(t, err)
	})
}

func TestCatalogService_DeleteProduct(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*MockRepository, CommandService) {
		repo := NewMockRepository()
		svc := NewCatalogCommandService(repo, &MockCacheManager{})
		return repo, svc
	}

	t.Run("successful product deletion", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		p := &domain.Product{ID: "prod_1", Name: "Smartphone"}
		repo.products[p.ID] = p

		err := svc.DeleteProduct(context.Background(), "prod_1")
		require.NoError(t, err)

		_, exists := repo.products["prod_1"]
		assert.False(t, exists)
	})
}

func TestCatalogService_AddCategory(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*MockRepository, CommandService) {
		repo := NewMockRepository()
		svc := NewCatalogCommandService(repo, &MockCacheManager{})
		return repo, svc
	}

	t.Run("successful category creation", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		c := &domain.Category{Name: "Home Appliances"}
		err := svc.AddCategory(context.Background(), c)
		require.NoError(t, err)
		assert.NotEmpty(t, c.ID)
		assert.Equal(t, "home-appliances", c.Slug)

		stored, err := repo.GetCategoryByID(context.Background(), c.ID)
		require.NoError(t, err)
		assert.Equal(t, "Home Appliances", stored.Name)
	})

	t.Run("fails - non-existent parent category", func(t *testing.T) {
		t.Parallel()
		_, svc := setup(t)

		parentID := "cat_nonexistent"
		c := &domain.Category{
			Name:     "Kitchen Appliances",
			ParentID: &parentID,
		}
		err := svc.AddCategory(context.Background(), c)
		assert.Error(t, err)
	})
}

func TestCatalogService_DeleteCategory(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*MockRepository, CommandService) {
		repo := NewMockRepository()
		svc := NewCatalogCommandService(repo, &MockCacheManager{})
		return repo, svc
	}

	t.Run("successful category deletion", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		c := &domain.Category{ID: "cat_1", Name: "Electronics"}
		repo.categories[c.ID] = c

		err := svc.DeleteCategory(context.Background(), "cat_1")
		require.NoError(t, err)

		_, exists := repo.categories["cat_1"]
		assert.False(t, exists)
	})

	t.Run("fails - assigned products", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		c := &domain.Category{ID: "cat_1", Name: "Electronics"}
		repo.categories[c.ID] = c

		p := &domain.Product{ID: "prod_1", CategoryID: "cat_1"}
		repo.products[p.ID] = p

		err := svc.DeleteCategory(context.Background(), "cat_1")
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 409, appErr.Code)
	})

	t.Run("fails - children subcategories exist", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		parentID := "cat_parent"
		parent := &domain.Category{ID: parentID, Name: "Electronics"}
		child := &domain.Category{ID: "cat_child", Name: "Smartphones", ParentID: &parentID}

		repo.categories[parent.ID] = parent
		repo.categories[child.ID] = child

		err := svc.DeleteCategory(context.Background(), parentID)
		assert.Error(t, err)
	})
}

func TestCatalogService_SubscribeStockEvents(t *testing.T) {
	t.Parallel()

	repo := NewMockRepository()
	bus := NewMockEventBus()
	svc := NewCatalogCommandService(repo, &MockCacheManager{})

	p := &domain.Product{
		ID:         "prod_1",
		Name:       "Test Product",
		CategoryID: "cat_1",
		Variants: []domain.Variant{
			{ID: "var_1", SKU: "TEST-SKU", Price: 10.00, Stock: 5},
		},
	}
	repo.products[p.ID] = p

	svc.SubscribeStockEvents(bus)

	// Publish stock changed event
	bus.Publish(events.Event{
		Topic: events.InventoryStockChangedTopic,
		Payload: &events.StockChangedPayload{
			VariantID:   "var_1",
			OldQuantity: 5,
			NewQuantity: 25,
		},
	})

	// Assert variant stock was synchronized to 25
	stored, err := repo.GetProductByID(context.Background(), "prod_1")
	require.NoError(t, err)
	assert.Equal(t, 25, stored.Variants[0].Stock)
}

func TestCatalogService_IDGenerationFailure(t *testing.T) {
	repo := NewMockRepository()
	// Instantiate service directly to override the idGenerator field
	svc := &catalogCommandService{
		repo: repo,
		idGenerator: func(_ string) (string, error) {
			return "", errors.New("entropy source exhausted")
		},
		cacheManager: &MockCacheManager{},
	}

	t.Run("AddProduct product ID generation failure", func(t *testing.T) {
		p := &domain.Product{
			Name:       "Smartphone",
			CategoryID: "cat_1",
		}
		err := svc.AddProduct(context.Background(), p)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 500, appErr.Code)
		assert.Contains(t, appErr.Error(), "failed to generate product ID")
	})

	t.Run("AddCategory ID generation failure", func(t *testing.T) {
		c := &domain.Category{Name: "Electronics"}
		err := svc.AddCategory(context.Background(), c)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 500, appErr.Code)
		assert.Contains(t, appErr.Error(), "failed to generate category ID")
	})
}

func TestCatalogService_ErrorMappingsAndCycles(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*MockRepository, QueryService, CommandService) {
		repo := NewMockRepository()
		qs := NewCatalogQueryService(repo, &MockCacheManager{})
		cs := NewCatalogCommandService(repo, &MockCacheManager{})
		return repo, qs, cs
	}

	t.Run("UpdateProduct - non-existent product returns 404", func(t *testing.T) {
		t.Parallel()
		repo, _, cs := setup(t)

		// Create category first so validation doesn't fail on category check
		cat := &domain.Category{ID: "cat_1", Name: "Electronics"}
		repo.categories[cat.ID] = cat

		pUpdate := &domain.Product{
			ID:         "prod_nonexistent",
			Name:       "New Smartphone",
			CategoryID: "cat_1",
			Variants: []domain.Variant{
				{ID: "var_1", SKU: "SM-PHN-01", Name: "Standard", Price: 749.99},
			},
		}

		repo.updateProdErr = ErrProductNotFound

		err := cs.UpdateProduct(context.Background(), pUpdate)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.Code)
	})

	t.Run("DeleteProduct - non-existent product returns 404", func(t *testing.T) {
		t.Parallel()
		_, _, cs := setup(t)

		err := cs.DeleteProduct(context.Background(), "prod_nonexistent")
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.Code)
	})

	t.Run("UpdateCategory - non-existent category returns 404", func(t *testing.T) {
		t.Parallel()
		repo, _, cs := setup(t)

		cUpdate := &domain.Category{
			ID:   "cat_nonexistent",
			Name: "Kitchen Appliances",
			Slug: "kitchen-appliances",
		}

		repo.updateCatErr = ErrCategoryNotFound

		err := cs.UpdateCategory(context.Background(), cUpdate)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.Code)
	})

	t.Run("DeleteCategory - non-existent category returns 404", func(t *testing.T) {
		t.Parallel()
		_, _, cs := setup(t)

		err := cs.DeleteCategory(context.Background(), "cat_nonexistent")
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.Code)
	})

	t.Run("UpdateCategory - circular dependency detection", func(t *testing.T) {
		t.Parallel()
		repo, _, cs := setup(t)

		parentID := "cat_parent"
		childID := "cat_child"

		parent := &domain.Category{ID: parentID, Name: "Electronics", Slug: "electronics"}
		child := &domain.Category{ID: childID, Name: "Smartphones", Slug: "smartphones", ParentID: &parentID}

		repo.categories[parentID] = parent
		repo.categories[childID] = child

		// Attempt to set parent's parent to child (creating parent -> child -> parent loop)
		parentUpdate := &domain.Category{
			ID:       parentID,
			Name:     "Electronics",
			Slug:     "electronics",
			ParentID: &childID,
		}

		err := cs.UpdateCategory(context.Background(), parentUpdate)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circular category dependency detected")
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.Code)
	})

	t.Run("UpdateProduct - version conflict returns 409", func(t *testing.T) {
		t.Parallel()
		repo, _, cs := setup(t)

		cat := &domain.Category{ID: "cat_1", Name: "Electronics"}
		repo.categories[cat.ID] = cat

		p := &domain.Product{
			ID:         "prod_1",
			Name:       "Old Smartphone",
			CategoryID: "cat_1",
			Variants: []domain.Variant{
				{ID: "var_1", SKU: "SM-PHN-01", Name: "Standard", Price: 699.99},
			},
			Version: 1,
		}
		repo.products[p.ID] = p

		pUpdate := &domain.Product{
			ID:         "prod_1",
			Name:       "New Smartphone",
			CategoryID: "cat_1",
			Variants: []domain.Variant{
				{ID: "var_1", SKU: "SM-PHN-01", Name: "Standard", Price: 749.99},
			},
			Version: 0,
		}

		err := cs.UpdateProduct(context.Background(), pUpdate)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 409, appErr.Code)
	})

	t.Run("AddCategory - category slug uniqueness validation", func(t *testing.T) {
		t.Parallel()
		repo, _, cs := setup(t)

		c1 := &domain.Category{ID: "cat_1", Name: "Electronics", Slug: "electronics"}
		repo.categories[c1.ID] = c1

		c2 := &domain.Category{Name: "Electronics Again", Slug: "electronics"}
		err := cs.AddCategory(context.Background(), c2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "electronics")
		assert.Contains(t, err.Error(), "already taken")
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 409, appErr.Code)
	})
}
