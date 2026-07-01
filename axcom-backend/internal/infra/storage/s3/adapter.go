// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package s3

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

type S3Adapter struct { //nolint:revive // Name is intentionally explicit for the public API.
	client        *s3.Client
	presignClient *s3.PresignClient
	region        string
}

// NewS3Adapter initialises a real AWS S3 client.
// If accessKeyID is non-empty, static credentials are used.
// Otherwise the standard AWS credential chain is used (env vars, IAM roles).
func NewS3Adapter(ctx context.Context, region, accessKeyID, secretAccessKey string) (*S3Adapter, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}
	if accessKeyID != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("s3: load config: %w", err)
	}

	client := s3.NewFromConfig(cfg)
	return &S3Adapter{
		client:        client,
		presignClient: s3.NewPresignClient(client),
		region:        region,
	}, nil
}

func (a *S3Adapter) Upload(ctx context.Context, bucket, key string, data io.Reader) (string, error) {
	_, err := a.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   data,
	})
	if err != nil {
		return "", fmt.Errorf("s3: upload %s/%s: %w", bucket, key, err)
	}
	return a.GetPublicURL(ctx, bucket, key)
}

func (a *S3Adapter) Download(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	out, err := a.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("s3: download %s/%s: %w", bucket, key, err)
	}
	return out.Body, nil
}

func (a *S3Adapter) Delete(ctx context.Context, bucket, key string) error {
	_, err := a.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
}

func (a *S3Adapter) PresignUpload(ctx context.Context, bucket, key, contentType string) (*storage.PresignUploadResult, error) {
	req, err := a.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(15*time.Minute))
	if err != nil {
		return nil, fmt.Errorf("s3: presign %s/%s: %w", bucket, key, err)
	}
	publicURL, _ := a.GetPublicURL(ctx, bucket, key)
	return &storage.PresignUploadResult{
		UploadURL: req.URL,
		PublicURL: publicURL,
		Method:    req.Method,
	}, nil
}

func (a *S3Adapter) GetPublicURL(_ context.Context, bucket, key string) (string, error) {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, a.region, key), nil
}
