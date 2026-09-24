// Package storage keeps uploaded media in S3-compatible storage (Garage in our infra).
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/bookly-kbtu/backend/internal/domain"
)

type Config struct {
	Endpoint        string // host:port, without scheme
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	UseSSL          bool
}

type S3 struct {
	client *minio.Client
	bucket string
}

func NewS3(ctx context.Context, cfg Config) (*S3, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}
	ok, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket %q: %w", cfg.Bucket, err)
	}
	if !ok {
		return nil, fmt.Errorf("bucket %q does not exist", cfg.Bucket)
	}
	return &S3{client: client, bucket: cfg.Bucket}, nil
}

func (s *S3) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, body, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("put %s: %w", key, err)
	}
	return nil
}

func (s *S3) Get(ctx context.Context, key string) (*domain.MediaObject, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", key, err)
	}
	// GetObject is lazy: Stat performs the request and reports a missing key.
	info, err := obj.Stat()
	if err != nil {
		_ = obj.Close()
		var resp minio.ErrorResponse
		if errors.As(err, &resp) && resp.Code == "NoSuchKey" {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("stat %s: %w", key, err)
	}
	return &domain.MediaObject{Body: obj, ContentType: info.ContentType, ContentLength: info.Size, ETag: info.ETag}, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete %s: %w", key, err)
	}
	return nil
}
