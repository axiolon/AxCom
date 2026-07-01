// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package images

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"ecom-engine/internal/core/catalog/domain"
	"ecom-engine/internal/infra/storage"
	apperrors "ecom-engine/pkg/errors"
	"ecom-engine/pkg/idgen"
	"ecom-engine/pkg/logger"
)

// Service defines the business contract for product images.
type Service interface {
	PresignUploadURLs(ctx context.Context, productID string, files []PresignImageRequest) ([]PresignImageResponse, error)
	RegisterUploadedImages(ctx context.Context, productID string, images []RegisterImageRequest) ([]domain.ProductImage, error)
	DeleteImage(ctx context.Context, productID string, imageID string) error
	SetPrimaryImage(ctx context.Context, productID string, imageID string) error
}

type imageService struct {
	repo    Repository
	storage storage.FileStorage
}

// NewService creates a new instance of the image service.
func NewService(repo Repository, storage storage.FileStorage) Service {
	return &imageService{repo: repo, storage: storage}
}

func (s *imageService) PresignUploadURLs(ctx context.Context, productID string, files []PresignImageRequest) ([]PresignImageResponse, error) {
	_, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return nil, apperrors.NewNotFound(fmt.Sprintf("product %s not found", productID), err)
	}

	if len(files) == 0 {
		return nil, apperrors.NewBadRequest("no files provided for presign request", nil)
	}

	var uploads []PresignImageResponse
	for _, file := range files {
		// Sanitize filename and extract extension
		baseName := filepath.Base(file.Filename)
		ext := strings.ToLower(filepath.Ext(baseName))

		// Validate extension
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" && ext != ".svg" {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("unsupported image extension: %s", ext), nil)
		}

		// Validate content type
		contentType := strings.ToLower(file.ContentType)
		if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/gif" && contentType != "image/webp" && contentType != "image/svg+xml" {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("unsupported image content type: %s", contentType), nil)
		}

		imgID, err := idgen.Generate("img_")
		if err != nil {
			return nil, apperrors.NewInternal("failed to generate image ID", err)
		}
		safeName := fmt.Sprintf("%s%s", imgID, ext)
		key := path.Join("products", productID, safeName)

		result, err := s.storage.PresignUpload(ctx, "products", key, file.ContentType)
		if err != nil {
			return nil, apperrors.NewInternal("failed to generate presigned upload URL", err)
		}

		uploads = append(uploads, PresignImageResponse{
			Filename:  file.Filename,
			UploadURL: result.UploadURL,
			PublicURL: result.PublicURL,
			Key:       key,
			Method:    result.Method,
		})
	}

	return uploads, nil
}

func (s *imageService) RegisterUploadedImages(ctx context.Context, productID string, images []RegisterImageRequest) ([]domain.ProductImage, error) {
	p, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return nil, apperrors.NewNotFound(fmt.Sprintf("product %s not found", productID), err)
	}

	if len(images) == 0 {
		return nil, apperrors.NewBadRequest("no uploaded images provided", nil)
	}

	var newImages []domain.ProductImage
	expectedPrefix := fmt.Sprintf("products/%s/", productID)
	for _, image := range images {
		// Strict path validation to prevent path traversal or referencing foreign products
		if !strings.HasPrefix(image.Key, expectedPrefix) || strings.Contains(image.Key, "..") {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("invalid image key: %s", image.Key), nil)
		}

		publicURL, err := s.storage.GetPublicURL(ctx, "products", image.Key)
		if err != nil {
			return nil, apperrors.NewInternal("failed to resolve public image URL", err)
		}

		imgID, err := idgen.Generate("img_")
		if err != nil {
			return nil, apperrors.NewInternal("failed to generate image ID", err)
		}

		newImages = append(newImages, domain.ProductImage{
			ID:        imgID,
			URL:       publicURL,
			Key:       image.Key,
			IsPrimary: image.IsPrimary,
		})
	}

	if len(p.Images) == 0 && len(newImages) > 0 {
		newImages[0].IsPrimary = true
	}

	p.Images = append(p.Images, newImages...)
	if err := s.repo.UpdateProductImages(ctx, productID, p.Images); err != nil {
		return nil, apperrors.NewInternal("failed to update product record with registered images", err)
	}

	return newImages, nil
}

func (s *imageService) DeleteImage(ctx context.Context, productID string, imageID string) error {
	p, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return apperrors.NewNotFound(fmt.Sprintf("product %s not found", productID), err)
	}

	var targetImage *domain.ProductImage
	var remainingImages []domain.ProductImage

	for i := range p.Images {
		if p.Images[i].ID == imageID {
			targetImage = &p.Images[i]
		} else {
			remainingImages = append(remainingImages, p.Images[i])
		}
	}

	if targetImage == nil {
		return apperrors.NewNotFound(fmt.Sprintf("image %s not found on product", imageID), nil)
	}

	deleteKey := targetImage.Key
	if deleteKey == "" {
		deleteKey = strings.TrimPrefix(strings.TrimPrefix(targetImage.URL, "/uploads/products/"), "/")
	}

	if err := s.storage.Delete(ctx, "products", deleteKey); err != nil {
		logger.DebugCtx(ctx, "Ignoring image delete error: %v", err)
	}

	if targetImage.IsPrimary && len(remainingImages) > 0 {
		remainingImages[0].IsPrimary = true
	}

	if err := s.repo.UpdateProductImages(ctx, productID, remainingImages); err != nil {
		return apperrors.NewInternal("failed to delete image from product record", err)
	}

	return nil
}

func (s *imageService) SetPrimaryImage(ctx context.Context, productID string, imageID string) error {
	p, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		return apperrors.NewNotFound(fmt.Sprintf("product %s not found", productID), err)
	}

	found := false
	for i := range p.Images {
		if p.Images[i].ID == imageID {
			p.Images[i].IsPrimary = true
			found = true
		} else {
			p.Images[i].IsPrimary = false
		}
	}

	if !found {
		return apperrors.NewNotFound(fmt.Sprintf("image %s not found on product", imageID), nil)
	}

	if err := s.repo.UpdateProductImages(ctx, productID, p.Images); err != nil {
		return apperrors.NewInternal("failed to update primary image setting", err)
	}

	return nil
}
