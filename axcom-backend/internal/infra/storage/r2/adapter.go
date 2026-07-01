// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package r2

import (
	"context"
	"fmt"
	"io"
	"time"

	"ecom-engine/internal/infra/storage"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Adapter struct { //nolint:revive // Name is intentionally explicit for the public API.
	client        *s3.Client
	presignClient *s3.PresignClient
	accountID     string
}

// NewR2Adapter initialises an S3-compatible client pointing to Cloudflare R2.
// accessKeyID and secretAccessKey are the R2 API token credentials.
func NewR2Adapter(ctx context.Context, accountID, accessKeyID, secretAccessKey string) (*R2Adapter, error) {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("auto"), // R2 expects region "auto"
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("r2: load config: %w", err)
	}

	r2Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	return &R2Adapter{
		client:        r2Client,
		presignClient: s3.NewPresignClient(r2Client),
		accountID:     accountID,
	}, nil
}

func (a *R2Adapter) Upload(ctx context.Context, bucket, key string, data io.Reader) (string, error) {
	_, err := a.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   data,
	})
	if err != nil {
		return "", fmt.Errorf("r2: upload %s/%s: %w", bucket, key, err)
	}
	return a.GetPublicURL(ctx, bucket, key)
}

func (a *R2Adapter) Download(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	out, err := a.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("r2: download %s/%s: %w", bucket, key, err)
	}
	return out.Body, nil
}

func (a *R2Adapter) Delete(ctx context.Context, bucket, key string) error {
	_, err := a.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
}

func (a *R2Adapter) PresignUpload(ctx context.Context, bucket, key, contentType string) (*storage.PresignUploadResult, error) {
	req, err := a.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(15*time.Minute))
	if err != nil {
		return nil, fmt.Errorf("r2: presign %s/%s: %w", bucket, key, err)
	}
	publicURL, _ := a.GetPublicURL(ctx, bucket, key)
	return &storage.PresignUploadResult{
		UploadURL: req.URL,
		PublicURL: publicURL,
		Method:    req.Method,
	}, nil
}

func (a *R2Adapter) GetPublicURL(_ context.Context, _, key string) (string, error) {
	return fmt.Sprintf("https://pub-%s.r2.dev/%s", a.accountID, key), nil
}
