// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package images

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"ecom-engine/internal/core/catalog/domain"
	"ecom-engine/internal/infra/storage"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockService struct {
	presignUploadURLs      func(ctx context.Context, productID string, files []PresignImageRequest) ([]PresignImageResponse, error)
	registerUploadedImages func(ctx context.Context, productID string, images []RegisterImageRequest) ([]domain.ProductImage, error)
	deleteImage            func(ctx context.Context, productID string, imageID string) error
	setPrimaryImage        func(ctx context.Context, productID string, imageID string) error
}

func (m *mockService) PresignUploadURLs(ctx context.Context, productID string, files []PresignImageRequest) ([]PresignImageResponse, error) {
	if m.presignUploadURLs != nil {
		return m.presignUploadURLs(ctx, productID, files)
	}
	return nil, nil
}

func (m *mockService) RegisterUploadedImages(ctx context.Context, productID string, images []RegisterImageRequest) ([]domain.ProductImage, error) {
	if m.registerUploadedImages != nil {
		return m.registerUploadedImages(ctx, productID, images)
	}
	return nil, nil
}

func (m *mockService) DeleteImage(ctx context.Context, productID string, imageID string) error {
	if m.deleteImage != nil {
		return m.deleteImage(ctx, productID, imageID)
	}
	return nil
}

func (m *mockService) SetPrimaryImage(ctx context.Context, productID string, imageID string) error {
	if m.setPrimaryImage != nil {
		return m.setPrimaryImage(ctx, productID, imageID)
	}
	return nil
}

type mockE2ERepository struct {
	products map[string]*domain.Product
}

func (m *mockE2ERepository) GetProductByID(_ context.Context, id string) (*domain.Product, error) {
	p, ok := m.products[id]
	if !ok {
		return nil, errors.New("product not found")
	}
	return p, nil
}

func (m *mockE2ERepository) UpdateProductImages(_ context.Context, id string, images []domain.ProductImage) error {
	p, ok := m.products[id]
	if !ok {
		return errors.New("product not found")
	}
	p.Images = images
	return nil
}

type mockE2EStorage struct{}

func (m *mockE2EStorage) Upload(_ context.Context, bucket, key string, _ io.Reader) (string, error) {
	return "http://storage/" + bucket + "/" + key, nil
}

func (m *mockE2EStorage) Download(_ context.Context, _, _ string) (io.ReadCloser, error) {
	return nil, nil
}

func (m *mockE2EStorage) Delete(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockE2EStorage) PresignUpload(_ context.Context, _, key, _ string) (*storage.PresignUploadResult, error) {
	return &storage.PresignUploadResult{
		UploadURL: "http://presigned/" + key,
		PublicURL: "http://storage/" + key,
		Method:    "PUT",
	}, nil
}

func (m *mockE2EStorage) GetPublicURL(_ context.Context, bucket, key string) (string, error) {
	return "http://storage/" + bucket + "/" + key, nil
}

func setupTestRouter(svc Service) *gin.Engine {
	router := gin.New()
	rg := router.Group("/api")
	dummyMiddleware := func(c *gin.Context) {
		c.Next()
	}
	ctrl := NewController(svc)
	RegisterRoutes(rg, ctrl, dummyMiddleware, dummyMiddleware)
	return router
}

// ----------------------------------------------------
// INTEGRATION TESTS (Mocked Service)
// ----------------------------------------------------

func TestController_PresignUpload_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful presign request", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			presignUploadURLs: func(_ context.Context, productID string, files []PresignImageRequest) ([]PresignImageResponse, error) {
				assert.Equal(t, "prod_1", productID)
				assert.Len(t, files, 1)
				return []PresignImageResponse{
					{Filename: "a.jpg", UploadURL: "http://upload-here"},
				}, nil
			},
		}

		router := setupTestRouter(mockSvc)
		reqBody := PresignImagesRequest{
			Files: []PresignImageRequest{
				{Filename: "a.jpg", ContentType: "image/jpeg"},
			},
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/products/prod_1/images/presign", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))
	})

	t.Run("fails - bad request body", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{}
		router := setupTestRouter(mockSvc)

		req, _ := http.NewRequest(http.MethodPost, "/api/products/prod_1/images/presign", bytes.NewBufferString("{bad_json}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestController_RegisterUploadedImages_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful registration", func(t *testing.T) {
		t.Parallel()
		mockSvc := &mockService{
			registerUploadedImages: func(_ context.Context, productID string, _ []RegisterImageRequest) ([]domain.ProductImage, error) {
				assert.Equal(t, "prod_1", productID)
				return []domain.ProductImage{
					{ID: "img_1", URL: "http://storage/img.jpg"},
				}, nil
			},
		}

		router := setupTestRouter(mockSvc)
		reqBody := RegisterImagesRequest{
			Images: []RegisterImageRequest{
				{Key: "products/prod_1/img.jpg", IsPrimary: true},
			},
		}
		jsonBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/products/prod_1/images/register", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ----------------------------------------------------
// END-TO-END TESTS (Full Controller + Service + DB Stack)
// ----------------------------------------------------

func TestImageFlow_E2E(t *testing.T) {
	t.Parallel()

	repo := &mockE2ERepository{
		products: map[string]*domain.Product{
			"prod_xyz": {
				ID:         "prod_xyz",
				Name:       "E2E Product",
				CategoryID: "cat_gadgets",
				Images:     []domain.ProductImage{},
			},
		},
	}
	storage := &mockE2EStorage{}
	service := NewService(repo, storage)
	router := setupTestRouter(service)

	// Verify initially no images
	p, err := repo.GetProductByID(context.Background(), "prod_xyz")
	require.NoError(t, err)
	assert.Empty(t, p.Images)

	// 1. POST to Presign URL
	presignReq := PresignImagesRequest{
		Files: []PresignImageRequest{
			{Filename: "thumbnail.png", ContentType: "image/png"},
		},
	}
	presignBytes, _ := json.Marshal(presignReq)
	req1, _ := http.NewRequest(http.MethodPost, "/api/products/prod_xyz/images/presign", bytes.NewBuffer(presignBytes))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// 2. POST to Register Uploaded Images (register 2 images)
	registerReq := RegisterImagesRequest{
		Images: []RegisterImageRequest{
			{Key: "products/prod_xyz/img1.png", IsPrimary: true},
			{Key: "products/prod_xyz/img2.png", IsPrimary: false},
		},
	}
	registerBytes, _ := json.Marshal(registerReq)
	req2, _ := http.NewRequest(http.MethodPost, "/api/products/prod_xyz/images/register", bytes.NewBuffer(registerBytes))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// Verify images exist and first is primary
	p, _ = repo.GetProductByID(context.Background(), "prod_xyz")
	require.Len(t, p.Images, 2)
	imgID1 := p.Images[0].ID
	imgID2 := p.Images[1].ID
	assert.True(t, p.Images[0].IsPrimary)
	assert.False(t, p.Images[1].IsPrimary)

	// 3. PUT to Set primary image
	req3, _ := http.NewRequest(http.MethodPut, "/api/products/prod_xyz/images/"+imgID2+"/primary", nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)

	p, _ = repo.GetProductByID(context.Background(), "prod_xyz")
	assert.False(t, p.Images[0].IsPrimary)
	assert.True(t, p.Images[1].IsPrimary)

	// 4. DELETE to remove the first image
	req4, _ := http.NewRequest(http.MethodDelete, "/api/products/prod_xyz/images/"+imgID1, nil)
	w4 := httptest.NewRecorder()
	router.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusOK, w4.Code)

	p, _ = repo.GetProductByID(context.Background(), "prod_xyz")
	require.Len(t, p.Images, 1)
	assert.Equal(t, imgID2, p.Images[0].ID)
}
