// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"io"
)

type PresignUploadResult struct {
	UploadURL string
	PublicURL string
	Method    string
}

type FileStorage interface {
	Upload(ctx context.Context, bucket, key string, data io.Reader) (string, error)
	Download(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, bucket, key string) error
	PresignUpload(ctx context.Context, bucket, key, contentType string) (*PresignUploadResult, error)
	GetPublicURL(ctx context.Context, bucket, key string) (string, error)
}
