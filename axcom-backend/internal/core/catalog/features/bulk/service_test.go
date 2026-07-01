// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package bulk

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"ecom-engine/internal/core/catalog/domain"
	apperrors "ecom-engine/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepository struct {
	bulkCreate      func(ctx context.Context, products []*domain.Product) error
	bulkUpdate      func(ctx context.Context, products []*domain.Product) error
	bulkDelete      func(ctx context.Context, ids []string) error
	getCategoryByID func(ctx context.Context, id string) (*domain.Category, error)
}

func (m *mockRepository) BulkCreate(ctx context.Context, products []*domain.Product) error {
	if m.bulkCreate != nil {
		return m.bulkCreate(ctx, products)
	}
	return nil
}

func (m *mockRepository) BulkUpdate(ctx context.Context, products []*domain.Product) error {
	if m.bulkUpdate != nil {
		return m.bulkUpdate(ctx, products)
	}
	return nil
}

func (m *mockRepository) BulkDelete(ctx context.Context, ids []string) error {
	if m.bulkDelete != nil {
		return m.bulkDelete(ctx, ids)
	}
	return nil
}

func (m *mockRepository) GetCategoryByID(ctx context.Context, id string) (*domain.Category, error) {
	if m.getCategoryByID != nil {
		return m.getCategoryByID(ctx, id)
	}
	return &domain.Category{ID: id, Name: "Dummy Category"}, nil
}

func TestBulkService_BulkCreate(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*mockRepository, Service) {
		repo := &mockRepository{}
		svc := NewService(repo)
		return repo, svc
	}

	t.Run("successful bulk create", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getCategoryByID = func(_ context.Context, id string) (*domain.Category, error) {
			assert.Equal(t, "cat_1", id)
			return &domain.Category{ID: "cat_1", Name: "Electronics"}, nil
		}

		repo.bulkCreate = func(_ context.Context, products []*domain.Product) error {
			assert.Len(t, products, 2)
			assert.NotEmpty(t, products[0].ID)
			assert.NotEmpty(t, products[0].Variants[0].ID)
			assert.Equal(t, "Product 1", products[0].Name)
			assert.Equal(t, "Product 2", products[1].Name)
			return nil
		}

		products := []*domain.Product{
			{
				Name:        "Product 1",
				CategoryID:  "cat_1",
				Description: "Desc 1",
				Variants: []domain.Variant{
					{SKU: "SKU-1", Name: "V1", Price: 10.0},
				},
			},
			{
				Name:        "Product 2",
				CategoryID:  "cat_1",
				Description: "Desc 2",
				Variants: []domain.Variant{
					{SKU: "SKU-2", Name: "V2", Price: 20.0},
				},
			},
		}

		err := svc.BulkCreate(context.Background(), products)
		require.NoError(t, err)
	})

	t.Run("fails - validation error empty product name", func(t *testing.T) {
		t.Parallel()
		_, svc := setup(t)

		products := []*domain.Product{
			{
				Name:       "",
				CategoryID: "cat_1",
				Variants: []domain.Variant{
					{SKU: "SKU-1", Name: "V1", Price: 10.0},
				},
			},
		}

		err := svc.BulkCreate(context.Background(), products)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.Code)
		assert.Contains(t, appErr.Error(), "validation failed")
	})

	t.Run("fails - category not found", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getCategoryByID = func(_ context.Context, _ string) (*domain.Category, error) {
			return nil, errors.New("category not found")
		}

		products := []*domain.Product{
			{
				Name:       "Product 1",
				CategoryID: "nonexistent",
				Variants: []domain.Variant{
					{SKU: "SKU-1", Name: "V1", Price: 10.0},
				},
			},
		}

		err := svc.BulkCreate(context.Background(), products)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.Code)
	})

	t.Run("fails - bulk create repo error", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.bulkCreate = func(_ context.Context, _ []*domain.Product) error {
			return errors.New("db error")
		}

		products := []*domain.Product{
			{
				Name:       "Product 1",
				CategoryID: "cat_1",
				Variants: []domain.Variant{
					{SKU: "SKU-1", Name: "V1", Price: 10.0},
				},
			},
		}

		err := svc.BulkCreate(context.Background(), products)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 500, appErr.Code)
	})
}

func TestBulkService_BulkUpdate(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*mockRepository, Service) {
		repo := &mockRepository{}
		svc := NewService(repo)
		return repo, svc
	}

	t.Run("successful bulk update", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.bulkUpdate = func(_ context.Context, products []*domain.Product) error {
			assert.Len(t, products, 1)
			assert.Equal(t, "prod_1", products[0].ID)
			assert.Equal(t, "Product Updated", products[0].Name)
			return nil
		}

		products := []*domain.Product{
			{
				ID:         "prod_1",
				Name:       "Product Updated",
				CategoryID: "cat_1",
				Variants: []domain.Variant{
					{ID: "var_1", SKU: "SKU-1", Name: "V1", Price: 15.0},
				},
			},
		}

		err := svc.BulkUpdate(context.Background(), products)
		require.NoError(t, err)
	})

	t.Run("fails - missing product ID", func(t *testing.T) {
		t.Parallel()
		_, svc := setup(t)

		products := []*domain.Product{
			{
				ID:         "",
				Name:       "Product Updated",
				CategoryID: "cat_1",
				Variants: []domain.Variant{
					{ID: "var_1", SKU: "SKU-1", Name: "V1", Price: 15.0},
				},
			},
		}

		err := svc.BulkUpdate(context.Background(), products)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.Code)
		assert.Contains(t, appErr.Error(), "product ID is required for update")
	})

	t.Run("fails - validation error negative price", func(t *testing.T) {
		t.Parallel()
		_, svc := setup(t)

		products := []*domain.Product{
			{
				ID:         "prod_1",
				Name:       "Product Updated",
				CategoryID: "cat_1",
				Variants: []domain.Variant{
					{ID: "var_1", SKU: "SKU-1", Name: "V1", Price: -5.0},
				},
			},
		}

		err := svc.BulkUpdate(context.Background(), products)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.Code)
		assert.Contains(t, appErr.Error(), "validation failed")
	})

	t.Run("fails - category not found during update", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getCategoryByID = func(_ context.Context, id string) (*domain.Category, error) {
			return nil, fmt.Errorf("category %s not found", id)
		}

		products := []*domain.Product{
			{
				ID:         "prod_1",
				Name:       "Product Updated",
				CategoryID: "nonexistent",
				Variants: []domain.Variant{
					{ID: "var_1", SKU: "SKU-1", Name: "V1", Price: 15.0},
				},
			},
		}

		err := svc.BulkUpdate(context.Background(), products)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.Code)
	})

	t.Run("fails - bulk update repo error", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.bulkUpdate = func(_ context.Context, _ []*domain.Product) error {
			return errors.New("db write failed")
		}

		products := []*domain.Product{
			{
				ID:         "prod_1",
				Name:       "Product Updated",
				CategoryID: "cat_1",
				Variants: []domain.Variant{
					{ID: "var_1", SKU: "SKU-1", Name: "V1", Price: 15.0},
				},
			},
		}

		err := svc.BulkUpdate(context.Background(), products)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 500, appErr.Code)
	})
}

func TestBulkService_BulkDelete(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*mockRepository, Service) {
		repo := &mockRepository{}
		svc := NewService(repo)
		return repo, svc
	}

	t.Run("successful bulk delete", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.bulkDelete = func(_ context.Context, ids []string) error {
			assert.Equal(t, []string{"prod_1", "prod_2"}, ids)
			return nil
		}

		err := svc.BulkDelete(context.Background(), []string{"prod_1", "prod_2"})
		require.NoError(t, err)
	})

	t.Run("fails - empty product IDs list", func(t *testing.T) {
		t.Parallel()
		_, svc := setup(t)

		err := svc.BulkDelete(context.Background(), []string{})
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.Code)
	})

	t.Run("fails - bulk delete repo error", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.bulkDelete = func(_ context.Context, _ []string) error {
			return errors.New("db delete failed")
		}

		err := svc.BulkDelete(context.Background(), []string{"prod_1"})
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 500, appErr.Code)
	})
}
