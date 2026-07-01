// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"ecom-engine/internal/infra/storage"
)

type LocalAdapter struct{} //nolint:revive // Name is intentionally explicit for the public API.

func NewLocalAdapter() *LocalAdapter {
	return &LocalAdapter{}
}

func (a *LocalAdapter) safePath(bucket, key string) (string, error) {
	basePath := filepath.Clean(filepath.Join(".", "uploads"))
	filePath := filepath.Clean(filepath.Join(basePath, bucket, filepath.FromSlash(key)))
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(absPath, absBase) {
		return "", fmt.Errorf("path traversal: path %s escapes base %s", absPath, absBase)
	}
	return filePath, nil
}

func (a *LocalAdapter) Upload(_ context.Context, bucket, key string, data io.Reader) (string, error) {
	filePath, err := a.safePath(bucket, key)
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(filepath.Dir(filePath), 0750); err != nil {
		return "", err
	}
	// #nosec G304 -- filePath is validated by safePath
	outFile, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = outFile.Close() }()

	if _, err := io.Copy(outFile, data); err != nil {
		return "", err
	}

	publicPath := fmt.Sprintf("/uploads/%s/%s", bucket, strings.ReplaceAll(key, "\\", "/"))
	return publicPath, nil
}

func (a *LocalAdapter) Download(_ context.Context, bucket, key string) (io.ReadCloser, error) {
	filePath, err := a.safePath(bucket, key)
	if err != nil {
		return nil, err
	}
	// #nosec G304 -- filePath is validated by safePath
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (a *LocalAdapter) Delete(_ context.Context, bucket, key string) error {
	filePath, err := a.safePath(bucket, key)
	if err != nil {
		return err
	}
	return os.Remove(filePath)
}

func (a *LocalAdapter) PresignUpload(_ context.Context, _, _, _ string) (*storage.PresignUploadResult, error) {
	return nil, errors.New("presign upload is not supported for local storage")
}

func (a *LocalAdapter) GetPublicURL(_ context.Context, bucket, key string) (string, error) {
	publicPath := fmt.Sprintf("/uploads/%s/%s", bucket, strings.ReplaceAll(key, "\\", "/"))
	return publicPath, nil
}
