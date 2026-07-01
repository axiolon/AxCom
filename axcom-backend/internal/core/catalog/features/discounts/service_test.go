// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package discounts

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
	updateProductDiscount func(ctx context.Context, id string, discount *domain.ProductDiscount) error
}

func (m *mockRepository) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	if m.getProductByID != nil {
		return m.getProductByID(ctx, id)
	}
	return nil, nil
}

func (m *mockRepository) UpdateProductDiscount(ctx context.Context, id string, discount *domain.ProductDiscount) error {
	if m.updateProductDiscount != nil {
		return m.updateProductDiscount(ctx, id, discount)
	}
	return nil
}

func TestDiscountService_ApplyDiscount(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*mockRepository, Service) {
		repo := &mockRepository{}
		svc := NewService(repo)
		return repo, svc
	}

	t.Run("successful percentage discount", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getProductByID = func(_ context.Context, id string) (*domain.Product, error) {
			assert.Equal(t, "prod_1", id)
			return &domain.Product{ID: "prod_1", Name: "Smartphone"}, nil
		}

		repo.updateProductDiscount = func(_ context.Context, id string, discount *domain.ProductDiscount) error {
			assert.Equal(t, "prod_1", id)
			assert.Equal(t, "percentage", discount.Type)
			assert.Equal(t, 15.5, discount.Value)
			return nil
		}

		d := &domain.ProductDiscount{Type: "percentage", Value: 15.5}
		err := svc.ApplyDiscount(context.Background(), "prod_1", d)
		require.NoError(t, err)
	})

	t.Run("successful fixed discount", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return &domain.Product{ID: "prod_2", Name: "Laptop"}, nil
		}

		repo.updateProductDiscount = func(_ context.Context, id string, discount *domain.ProductDiscount) error {
			assert.Equal(t, "prod_2", id)
			assert.Equal(t, "fixed", discount.Type)
			assert.Equal(t, 50.0, discount.Value)
			return nil
		}

		d := &domain.ProductDiscount{Type: "fixed", Value: 50.0}
		err := svc.ApplyDiscount(context.Background(), "prod_2", d)
		require.NoError(t, err)
	})

	t.Run("fails - nil discount payload", func(t *testing.T) {
		t.Parallel()
		_, svc := setup(t)

		err := svc.ApplyDiscount(context.Background(), "prod_1", nil)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.Code)
		assert.Contains(t, appErr.Error(), "Discount payload is required")
	})

	t.Run("fails - invalid discount type", func(t *testing.T) {
		t.Parallel()
		_, svc := setup(t)

		d := &domain.ProductDiscount{Type: "invalid_type", Value: 10}
		err := svc.ApplyDiscount(context.Background(), "prod_1", d)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.Code)
	})

	t.Run("fails - negative discount value", func(t *testing.T) {
		t.Parallel()
		_, svc := setup(t)

		d := &domain.ProductDiscount{Type: "fixed", Value: -5}
		err := svc.ApplyDiscount(context.Background(), "prod_1", d)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.Code)
	})

	t.Run("fails - percentage exceeding 100", func(t *testing.T) {
		t.Parallel()
		_, svc := setup(t)

		d := &domain.ProductDiscount{Type: "percentage", Value: 105}
		err := svc.ApplyDiscount(context.Background(), "prod_1", d)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.Code)
	})

	t.Run("fails - product not found", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return nil, errors.New("not found")
		}

		d := &domain.ProductDiscount{Type: "percentage", Value: 10}
		err := svc.ApplyDiscount(context.Background(), "prod_nonexistent", d)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.Code)
	})

	t.Run("fails - repository update failure", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return &domain.Product{ID: "prod_1", Name: "Smartphone"}, nil
		}

		repo.updateProductDiscount = func(_ context.Context, _ string, _ *domain.ProductDiscount) error {
			return errors.New("database connection lost")
		}

		d := &domain.ProductDiscount{Type: "percentage", Value: 10}
		err := svc.ApplyDiscount(context.Background(), "prod_1", d)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 500, appErr.Code)
	})
}

func TestDiscountService_RemoveDiscount(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*mockRepository, Service) {
		repo := &mockRepository{}
		svc := NewService(repo)
		return repo, svc
	}

	t.Run("successful remove discount", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return &domain.Product{ID: "prod_1", Name: "Smartphone"}, nil
		}

		repo.updateProductDiscount = func(_ context.Context, id string, discount *domain.ProductDiscount) error {
			assert.Equal(t, "prod_1", id)
			assert.Nil(t, discount)
			return nil
		}

		err := svc.RemoveDiscount(context.Background(), "prod_1")
		require.NoError(t, err)
	})

	t.Run("fails - product not found", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return nil, errors.New("not found")
		}

		err := svc.RemoveDiscount(context.Background(), "prod_nonexistent")
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 404, appErr.Code)
	})

	t.Run("fails - repository update failure", func(t *testing.T) {
		t.Parallel()
		repo, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return &domain.Product{ID: "prod_1", Name: "Smartphone"}, nil
		}

		repo.updateProductDiscount = func(_ context.Context, _ string, _ *domain.ProductDiscount) error {
			return errors.New("db error")
		}

		err := svc.RemoveDiscount(context.Background(), "prod_1")
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 500, appErr.Code)
	})
}
