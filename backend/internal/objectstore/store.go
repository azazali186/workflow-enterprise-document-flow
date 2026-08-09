// Package objectstore abstracts binary persistence behind a small interface.
// Production uses S3-compatible storage (minio-go); local disk is used in
// development when S3 is not configured.
package objectstore

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/config"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/retry"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/uuidx"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ObjectStore persists document binaries.
type ObjectStore interface {
	// Put stores reader content under key and returns the size written.
	Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (int64, error)
	// Get returns a reader for the object at key. The caller closes it.
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes the object at key.
	Delete(ctx context.Context, key string) error
	// Provider returns the storage provider name ("local" or "s3").
	Provider() string
	// Bucket returns the bucket/root name.
	Bucket() string
}

// LocalStore writes files under a local directory.
type LocalStore struct {
	Dir string
}

// Put implements ObjectStore. Document binaries are private to the service,
// so files are created 0600 (not the umask-derived 0666 default).
func (l *LocalStore) Put(_ context.Context, key string, r io.Reader, size int64, _ string) (int64, error) {
	if err := os.MkdirAll(l.Dir, 0o750); err != nil {
		return 0, err
	}
	//nolint:gosec // G304: key is server-generated or IsLocal-validated at registration
	dst, err := os.OpenFile(filepath.Join(l.Dir, key), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return 0, err
	}
	defer func() { _ = dst.Close() }()
	n, err := io.Copy(dst, r)
	if err != nil {
		_ = os.Remove(filepath.Join(l.Dir, key))
		return 0, err
	}
	return n, nil
}

// Get implements ObjectStore. The key is always a server-generated name or a
// client-supplied key validated as a safe relative path at registration
// (filepath.IsLocal), so the variable-path open cannot escape l.Dir.
func (l *LocalStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(l.Dir, key)) //nolint:gosec // G304: key is server-generated or IsLocal-validated
}

// Delete implements ObjectStore.
func (l *LocalStore) Delete(_ context.Context, key string) error {
	return os.Remove(filepath.Join(l.Dir, key)) //nolint:gosec // G304: key is server-generated or IsLocal-validated
}

// Provider implements ObjectStore.
func (l *LocalStore) Provider() string { return "local" }

// Bucket implements ObjectStore.
func (l *LocalStore) Bucket() string { return l.Dir }

// S3Store persists objects to an S3-compatible endpoint.
type S3Store struct {
	client *minio.Client
	bucket string
	region string
}

// Put implements ObjectStore (multipart-safe via minio-go).
func (s *S3Store) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) (int64, error) {
	if _, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	}); err != nil {
		return 0, err
	}
	return size, nil
}

// Get implements ObjectStore.
func (s *S3Store) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// Delete implements ObjectStore.
func (s *S3Store) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

// Provider implements ObjectStore.
func (s *S3Store) Provider() string { return "s3" }

// Bucket implements ObjectStore.
func (s *S3Store) Bucket() string { return s.bucket }

// New selects the object store from configuration: S3 when configured,
// otherwise the local directory.
func New(cfg *config.Config) (ObjectStore, error) {
	if cfg.S3Endpoint == "" || cfg.S3AccessKey == "" || cfg.S3SecretKey == "" {
		return &LocalStore{Dir: cfg.StorageDir}, nil
	}
	client, err := minio.New(cfg.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		Secure: cfg.S3UseSSL,
		Region: cfg.S3Region,
	})
	if err != nil {
		return nil, fmt.Errorf("s3 client: %w", err)
	}
	// Object storage (minio/S3) may still be booting when the app starts —
	// especially in CI where the whole stack comes up together. Retry the
	// bucket init for a short window instead of crash-looping on the race.
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := retry.Do(ctx, retry.Options{
		MaxAttempts: 12, BaseDelay: 500 * time.Millisecond, MaxDelay: 5 * time.Second, Factor: 1.5,
	}, func(attempt int) error {
		if err := ensureBucket(ctx, client, cfg.S3Bucket); err != nil {
			return fmt.Errorf("ensure bucket (attempt %d): %w", attempt, err)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("s3 bucket: %w", err)
	}
	return &S3Store{client: client, bucket: cfg.S3Bucket, region: cfg.S3Region}, nil
}

// NewKey generates a collision-safe object key for a file name.
func NewKey(fileName string) string {
	return uuidx.New() + "-" + Sanitize(fileName)
}

func ensureBucket(ctx context.Context, client *minio.Client, bucket string) error {
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
}
