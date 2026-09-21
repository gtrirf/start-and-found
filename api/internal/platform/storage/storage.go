// Package storage wraps the S3 compatible object storage that keeps uploaded
// media. PostgreSQL only stores metadata and references (see the media domain).
package storage

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config describes the bucket the API reads and writes.
type Config struct {
	Endpoint      string
	Region        string
	Bucket        string
	AccessKey     string
	SecretKey     string
	UseSSL        bool
	PublicBaseURL string
}

// Client is a thin wrapper around the MinIO client.
type Client struct {
	client        *minio.Client
	bucket        string
	publicBaseURL string
}

// New builds a client. The client is lazy, so a temporarily unreachable object
// store does not prevent the API from starting.
func New(cfg Config) (*Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create object storage client: %w", err)
	}
	return &Client{
		client:        client,
		bucket:        cfg.Bucket,
		publicBaseURL: strings.TrimRight(cfg.PublicBaseURL, "/"),
	}, nil
}

// Bucket returns the configured bucket name.
func (c *Client) Bucket() string { return c.bucket }

// PresignPut returns a time limited URL that a client may PUT a file to.
func (c *Client) PresignPut(ctx context.Context, key string, expiry time.Duration) (string, error) {
	signed, err := c.client.PresignedPutObject(ctx, c.bucket, key, expiry)
	if err != nil {
		return "", fmt.Errorf("presign upload: %w", err)
	}
	return signed.String(), nil
}

// PresignGet returns a time limited URL for reading a private object.
func (c *Client) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	signed, err := c.client.PresignedGetObject(ctx, c.bucket, key, expiry, url.Values{})
	if err != nil {
		return "", fmt.Errorf("presign download: %w", err)
	}
	return signed.String(), nil
}

// PublicURL builds the public URL of an object. It returns an empty string when
// no public base URL is configured.
func (c *Client) PublicURL(key string) string {
	if c.publicBaseURL == "" {
		return ""
	}
	return c.publicBaseURL + "/" + strings.TrimLeft(key, "/")
}

// Stat returns the size and content type of a stored object.
func (c *Client) Stat(ctx context.Context, key string) (int64, string, error) {
	info, err := c.client.StatObject(ctx, c.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return 0, "", fmt.Errorf("stat object %q: %w", key, err)
	}
	return info.Size, info.ContentType, nil
}

// Remove deletes a stored object.
func (c *Client) Remove(ctx context.Context, key string) error {
	if err := c.client.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("remove object %q: %w", key, err)
	}
	return nil
}

// EnsureBucket creates the bucket when it is missing.
func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.client.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("check bucket %q: %w", c.bucket, err)
	}
	if exists {
		return nil
	}
	if err := c.client.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("create bucket %q: %w", c.bucket, err)
	}
	return nil
}

// ObjectKey builds the storage key of an upload: users/<user>/<random>/<name>.
func ObjectKey(ownerID, mediaID, filename string) string {
	safeName := path.Base(strings.ReplaceAll(filename, "\\", "/"))
	return path.Join("users", ownerID, mediaID, safeName)
}
