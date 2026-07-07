package miniostore

import (
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func newClient(host string, port int) (*minio.Client, error) {
	return minio.New(fmt.Sprintf("%s:%d", host, port), &minio.Options{
		Creds:  credentials.NewStaticV4(RootUser, RootPassword, ""),
		Secure: false,
	})
}

func ListBuckets(ctx context.Context, host string, port int) ([]string, error) {
	c, err := newClient(host, port)
	if err != nil {
		return nil, err
	}
	buckets, err := c.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(buckets))
	for _, b := range buckets {
		out = append(out, b.Name)
	}
	return out, nil
}

type Object struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	LastModified string `json:"lastModified"`
}

func ListObjects(ctx context.Context, host string, port int, bucket, prefix string) ([]Object, error) {
	c, err := newClient(host, port)
	if err != nil {
		return nil, err
	}
	out := []Object{}
	for obj := range c.ListObjects(ctx, bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		if obj.Err != nil {
			return nil, obj.Err
		}
		out = append(out, Object{
			Key:          obj.Key,
			Size:         obj.Size,
			LastModified: obj.LastModified.Format(time.RFC3339),
		})
	}
	return out, nil
}

func PresignedURL(ctx context.Context, host string, port int, bucket, key string) (string, error) {
	c, err := newClient(host, port)
	if err != nil {
		return "", err
	}
	u, err := c.PresignedGetObject(ctx, bucket, key, 10*time.Minute, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
