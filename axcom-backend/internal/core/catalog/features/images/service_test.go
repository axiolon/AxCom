// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package images

import (
	"context"
	"errors"
	"io"
	"testing"

	"ecom-engine/internal/core/catalog/domain"
	"ecom-engine/internal/infra/storage"
	apperrors "ecom-engine/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepository struct {
	getProductByID      func(ctx context.Context, id string) (*domain.Product, error)
	updateProductImages func(ctx context.Context, id string, images []domain.ProductImage) error
}

func (m *mockRepository) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	if m.getProductByID != nil {
		return m.getProductByID(ctx, id)
	}
	return nil, nil
}

func (m *mockRepository) UpdateProductImages(ctx context.Context, id string, images []domain.ProductImage) error {
	if m.updateProductImages != nil {
		return m.updateProductImages(ctx, id, images)
	}
	return nil
}

type mockFileStorage struct {
	upload        func(ctx context.Context, bucket, key string, data io.Reader) (string, error)
	download      func(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	delete        func(ctx context.Context, bucket, key string) error
	presignUpload func(ctx context.Context, bucket, key, contentType string) (*storage.PresignUploadResult, error)
	getPublicURL  func(ctx context.Context, bucket, key string) (string, error)
}

func (m *mockFileStorage) Upload(ctx context.Context, bucket, key string, data io.Reader) (string, error) {
	if m.upload != nil {
		return m.upload(ctx, bucket, key, data)
	}
	return "", nil
}

func (m *mockFileStorage) Download(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	if m.download != nil {
		return m.download(ctx, bucket, key)
	}
	return nil, nil
}

func (m *mockFileStorage) Delete(ctx context.Context, bucket, key string) error {
	if m.delete != nil {
		return m.delete(ctx, bucket, key)
	}
	return nil
}

func (m *mockFileStorage) PresignUpload(ctx context.Context, bucket, key, contentType string) (*storage.PresignUploadResult, error) {
	if m.presignUpload != nil {
		return m.presignUpload(ctx, bucket, key, contentType)
	}
	return nil, nil
}

func (m *mockFileStorage) GetPublicURL(ctx context.Context, bucket, key string) (string, error) {
	if m.getPublicURL != nil {
		return m.getPublicURL(ctx, bucket, key)
	}
	return "", nil
}

func TestImageService_PresignUploadURLs(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*mockRepository, *mockFileStorage, Service) {
		repo := &mockRepository{}
		store := &mockFileStorage{}
		svc := NewService(repo, store)
		return repo, store, svc
	}

	t.Run("successful presign", func(t *testing.T) {
		t.Parallel()
		repo, store, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return &domain.Product{ID: "prod_1"}, nil
		}

		store.presignUpload = func(_ context.Context, _, _, _ string) (*storage.PresignUploadResult, error) {
			return &storage.PresignUploadResult{
				UploadURL: "http://presigned-upload-url",
				PublicURL: "http://public-url",
				Method:    "PUT",
			}, nil
		}

		reqs := []PresignImageRequest{
			{Filename: "photo.jpg", ContentType: "image/jpeg"},
		}

		resps, err := svc.PresignUploadURLs(context.Background(), "prod_1", reqs)
		require.NoError(t, err)
		assert.Len(t, resps, 1)
		assert.Equal(t, "http://presigned-upload-url", resps[0].UploadURL)
	})

	t.Run("fails - empty files", func(t *testing.T) {
		t.Parallel()
		repo, _, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return &domain.Product{ID: "prod_1"}, nil
		}

		_, err := svc.PresignUploadURLs(context.Background(), "prod_1", nil)
		assert.Error(t, err)
		var appErr *apperrors.AppError
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, 400, appErr.Code)
	})
}

func TestImageService_RegisterUploadedImages(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*mockRepository, *mockFileStorage, Service) {
		repo := &mockRepository{}
		store := &mockFileStorage{}
		svc := NewService(repo, store)
		return repo, store, svc
	}

	t.Run("successful register", func(t *testing.T) {
		t.Parallel()
		repo, store, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return &domain.Product{ID: "prod_1"}, nil
		}

		store.getPublicURL = func(_ context.Context, _ string, key string) (string, error) {
			return "http://public/" + key, nil
		}

		repo.updateProductImages = func(_ context.Context, _ string, images []domain.ProductImage) error {
			assert.Len(t, images, 1)
			assert.True(t, images[0].IsPrimary)
			return nil
		}

		imgs, err := svc.RegisterUploadedImages(context.Background(), "prod_1", []RegisterImageRequest{
			{Key: "products/prod_1/abc.jpg", IsPrimary: true},
		})
		require.NoError(t, err)
		assert.Len(t, imgs, 1)
		assert.Equal(t, "http://public/products/prod_1/abc.jpg", imgs[0].URL)
	})
}

func TestImageService_DeleteImage(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*mockRepository, *mockFileStorage, Service) {
		repo := &mockRepository{}
		store := &mockFileStorage{}
		svc := NewService(repo, store)
		return repo, store, svc
	}

	t.Run("successful delete image", func(t *testing.T) {
		t.Parallel()
		repo, store, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return &domain.Product{
				ID: "prod_1",
				Images: []domain.ProductImage{
					{ID: "img_1", URL: "/uploads/products/prod_1/img_1.jpg", IsPrimary: true},
					{ID: "img_2", URL: "/uploads/products/prod_1/img_2.jpg", IsPrimary: false},
				},
			}, nil
		}

		store.delete = func(_ context.Context, _ string, key string) error {
			assert.Equal(t, "prod_1/img_1.jpg", key)
			return nil
		}

		repo.updateProductImages = func(_ context.Context, _ string, images []domain.ProductImage) error {
			assert.Len(t, images, 1)
			assert.Equal(t, "img_2", images[0].ID)
			assert.True(t, images[0].IsPrimary) // should promote the remaining one
			return nil
		}

		err := svc.DeleteImage(context.Background(), "prod_1", "img_1")
		require.NoError(t, err)
	})
}

func TestImageService_SetPrimaryImage(t *testing.T) {
	t.Parallel()

	setup := func(_ *testing.T) (*mockRepository, *mockFileStorage, Service) {
		repo := &mockRepository{}
		store := &mockFileStorage{}
		svc := NewService(repo, store)
		return repo, store, svc
	}

	t.Run("successful set primary", func(t *testing.T) {
		t.Parallel()
		repo, _, svc := setup(t)

		repo.getProductByID = func(_ context.Context, _ string) (*domain.Product, error) {
			return &domain.Product{
				ID: "prod_1",
				Images: []domain.ProductImage{
					{ID: "img_1", URL: "url1", IsPrimary: true},
					{ID: "img_2", URL: "url2", IsPrimary: false},
				},
			}, nil
		}

		repo.updateProductImages = func(_ context.Context, _ string, images []domain.ProductImage) error {
			assert.False(t, images[0].IsPrimary)
			assert.True(t, images[1].IsPrimary)
			return nil
		}

		err := svc.SetPrimaryImage(context.Background(), "prod_1", "img_2")
		require.NoError(t, err)
	})
}
