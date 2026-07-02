package miniostore

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	RootUser     = "orbit"
	RootPassword = "orbitsecret"
)

func EnsureBucket(ctx context.Context, host string, port int, slug string) error {
	client, err := minio.New(fmt.Sprintf("%s:%d", host, port), &minio.Options{
		Creds:  credentials.NewStaticV4(RootUser, RootPassword, ""),
		Secure: false,
	})
	if err != nil {
		return fmt.Errorf("minio: client: %w", err)
	}
	exists, err := client.BucketExists(ctx, slug)
	if err != nil {
		return fmt.Errorf("minio: check bucket: %w", err)
	}
	if exists {
		return nil
	}
	if err := client.MakeBucket(ctx, slug, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("minio: create bucket: %w", err)
	}
	return nil
}
