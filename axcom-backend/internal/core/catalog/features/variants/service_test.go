// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package variants

import (
	"context"
	"errors"
	"testing"

	"ecom-engine/internal/core/catalog/domain"
	apperrors "ecom-engine/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepository struct {
	getProductByID        func(ctx context.Context, id string) (*domain.Product, error)
	updateProductVariants func(ctx context.Context, id string, variants []domain.Variant) error
}

func (m *mockRepository) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	if m.getProductByID != nil {
		return m.getProductByID(ctx, id)
	}
	return nil, nil
}

func (m *mockRepository) UpdateProductVariants(ctx context.Context, id string, variants []domain.Variant) error {
	if m.updateProductVariants != nil {
		return m.updateProductVariants(ctx, id, variants)
	}
	return nil
}

func TestVariantsService_GetVariants(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*mockRepository, Service) {
		repo := &mockRepository{}
		svc := NewService(repo)
		return repo, svc
	}

	t.Run("successful retrieval", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		expectedVariants := []domain.Variant{
			{ID: "var_1", SKU: "SKU-1", Name: "Variant 1", Price: 10.0},
		}

		repo.getProductByID = func(_ context.Context, id string) (*domain.Product, error) {
			assert.Equal(t, "prod_1", id)
			return &domain.Product{
				ID:       "prod_1",
				Name:     "Test Product",
				Variants: expectedVariants,
			}, nil
		}

		variants, err := svc.GetVariants(context.Background(), "prod_1")
		require.NoError(t, err)
		assert.Equal(t, expectedVariants, variants)
	})

	t.Run("product not found", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return nil, errors.New("db error")
		}

		variants, err := svc.GetVariants(context.Background(), "prod_nonexistent")
		assert.Nil(t, variants)
		assert.Error(t, err)

		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.Code)
		assert.Contains(t, appErr.Error(), "product prod_nonexistent not found")
	})
}

func TestVariantsService_AddVariant(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*mockRepository, Service) {
		repo := &mockRepository{}
		svc := NewService(repo)
		return repo, svc
	}

	t.Run("successful addition without ID", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return &domain.Product{
				ID: "prod_1",
				Variants: []domain.Variant{
					{ID: "var_1", SKU: "SKU-1", Name: "Variant 1", Price: 10.0},
				},
			}, nil
		}

		repo.updateProductVariants = func(_ context.Context, id string, variants []domain.Variant) error {
			assert.Equal(t, "prod_1", id)
			assert.Len(t, variants, 2)
			assert.NotEmpty(t, variants[1].ID)
			assert.Contains(t, variants[1].ID, "var_")
			assert.Equal(t, "SKU-2", variants[1].SKU)
			assert.Equal(t, 20.0, variants[1].Price)
			return nil
		}

		newVariant := &domain.Variant{
			SKU:   "SKU-2",
			Name:  "Variant 2",
			Price: 20.0,
		}

		err := svc.AddVariant(context.Background(), "prod_1", newVariant)
		require.NoError(t, err)
		assert.NotEmpty(t, newVariant.ID)
	})

	t.Run("product not found", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return nil, errors.New("db error")
		}

		newVariant := &domain.Variant{SKU: "SKU-1", Name: "V1", Price: 10.0}
		err := svc.AddVariant(context.Background(), "prod_nonexistent", newVariant)
		assert.Error(t, err)

		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.Code)
	})

	t.Run("validation error duplicate SKU", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return &domain.Product{
				ID: "prod_1",
				Variants: []domain.Variant{
					{ID: "var_1", SKU: "SKU-1", Name: "Variant 1", Price: 10.0},
				},
			}, nil
		}

		newVariant := &domain.Variant{
			SKU:   "SKU-1",
			Name:  "Duplicate SKU Variant",
			Price: 20.0,
		}

		err := svc.AddVariant(context.Background(), "prod_1", newVariant)
		assert.Error(t, err)

		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.Code)
		assert.Contains(t, appErr.Error(), "duplicate SKU")
	})

	t.Run("repo update error", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return &domain.Product{
				ID:       "prod_1",
				Variants: []domain.Variant{},
			}, nil
		}

		repo.updateProductVariants = func(_ context.Context, _ string, _ []domain.Variant) error {
			return errors.New("db update failed")
		}

		newVariant := &domain.Variant{SKU: "SKU-1", Name: "V1", Price: 10.0}
		err := svc.AddVariant(context.Background(), "prod_1", newVariant)
		assert.Error(t, err)

		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 500, appErr.Code)
		assert.Contains(t, appErr.Error(), "failed to add variant")
	})
}

func TestVariantsService_UpdateVariant(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*mockRepository, Service) {
		repo := &mockRepository{}
		svc := NewService(repo)
		return repo, svc
	}

	t.Run("successful update", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return &domain.Product{
				ID: "prod_1",
				Variants: []domain.Variant{
					{ID: "var_1", SKU: "SKU-1", Name: "Variant 1", Price: 10.0},
				},
			}, nil
		}

		repo.updateProductVariants = func(_ context.Context, id string, variants []domain.Variant) error {
			assert.Equal(t, "prod_1", id)
			assert.Len(t, variants, 1)
			assert.Equal(t, "var_1", variants[0].ID)
			assert.Equal(t, "SKU-UPDATED", variants[0].SKU)
			assert.Equal(t, 15.0, variants[0].Price)
			return nil
		}

		updatedVariant := &domain.Variant{
			ID:    "var_1",
			SKU:   "SKU-UPDATED",
			Name:  "Variant Updated",
			Price: 15.0,
		}

		err := svc.UpdateVariant(context.Background(), "prod_1", updatedVariant)
		require.NoError(t, err)
	})

	t.Run("missing variant ID", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return &domain.Product{
				ID: "prod_1",
				Variants: []domain.Variant{
					{ID: "var_1", SKU: "SKU-1", Name: "Variant 1", Price: 10.0},
				},
			}, nil
		}

		updatedVariant := &domain.Variant{
			SKU:   "SKU-UPDATED",
			Name:  "Variant Updated",
			Price: 15.0,
		}

		err := svc.UpdateVariant(context.Background(), "prod_1", updatedVariant)
		assert.Error(t, err)

		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.Code)
		assert.Contains(t, appErr.Error(), "Variant ID is required")
	})

	t.Run("variant not found", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return &domain.Product{
				ID: "prod_1",
				Variants: []domain.Variant{
					{ID: "var_1", SKU: "SKU-1", Name: "Variant 1", Price: 10.0},
				},
			}, nil
		}

		updatedVariant := &domain.Variant{
			ID:    "var_nonexistent",
			SKU:   "SKU-UPDATED",
			Name:  "Variant Updated",
			Price: 15.0,
		}

		err := svc.UpdateVariant(context.Background(), "prod_1", updatedVariant)
		assert.Error(t, err)

		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.Code)
		assert.Contains(t, appErr.Error(), "variant var_nonexistent not found")
	})
}

func TestVariantsService_DeleteVariant(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*mockRepository, Service) {
		repo := &mockRepository{}
		svc := NewService(repo)
		return repo, svc
	}

	t.Run("successful deletion", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return &domain.Product{
				ID: "prod_1",
				Variants: []domain.Variant{
					{ID: "var_1", SKU: "SKU-1", Name: "V1", Price: 10.0},
					{ID: "var_2", SKU: "SKU-2", Name: "V2", Price: 20.0},
				},
			}, nil
		}

		repo.updateProductVariants = func(_ context.Context, id string, variants []domain.Variant) error {
			assert.Equal(t, "prod_1", id)
			assert.Len(t, variants, 1)
			assert.Equal(t, "var_2", variants[0].ID)
			return nil
		}

		err := svc.DeleteVariant(context.Background(), "prod_1", "var_1")
		require.NoError(t, err)
	})

	t.Run("cannot delete last remaining variant", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return &domain.Product{
				ID: "prod_1",
				Variants: []domain.Variant{
					{ID: "var_1", SKU: "SKU-1", Name: "V1", Price: 10.0},
				},
			}, nil
		}

		err := svc.DeleteVariant(context.Background(), "prod_1", "var_1")
		assert.Error(t, err)

		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.Code)
		assert.Contains(t, appErr.Error(), "must have at least one variant")
	})

	t.Run("variant not found", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return &domain.Product{
				ID: "prod_1",
				Variants: []domain.Variant{
					{ID: "var_1", SKU: "SKU-1", Name: "V1", Price: 10.0},
					{ID: "var_2", SKU: "SKU-2", Name: "V2", Price: 20.0},
				},
			}, nil
		}

		err := svc.DeleteVariant(context.Background(), "prod_1", "var_nonexistent")
		assert.Error(t, err)

		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.Code)
		assert.Contains(t, appErr.Error(), "variant var_nonexistent not found")
	})

	t.Run("validation error empty SKU", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return &domain.Product{
				ID: "prod_1",
				Variants: []domain.Variant{
					{ID: "var_1", SKU: "SKU-1", Name: "Variant 1", Price: 10.0},
				},
			}, nil
		}

		newVariant := &domain.Variant{
			SKU:   "", // Empty SKU
			Name:  "Empty SKU Variant",
			Price: 20.0,
		}

		err := svc.AddVariant(context.Background(), "prod_1", newVariant)
		assert.Error(t, err)

		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.Code)
		assert.Contains(t, appErr.Error(), "SKU is required")
	})
}
