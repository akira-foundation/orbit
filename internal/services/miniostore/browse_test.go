package miniostore

import "testing"

func TestListBucketsUnreachable(t *testing.T) {
	if _, err := ListBuckets(t.Context(), "127.0.0.1", 1); err == nil {
		t.Fatal("expected error against dead endpoint")
	}
}
