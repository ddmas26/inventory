// Package storage keeps product images in an S3-compatible object store: MinIO
// during development and AWS S3 in production, with no code difference between
// the two. Only the configuration changes.
//
// References in the database are object keys (e.g. "products/9f2c….jpg") rather
// than full URLs, so the same rows keep working when the endpoint changes.
// Anything that is already an absolute URL or a site-relative path is stored and
// returned unchanged, which keeps external image links and files uploaded before
// the bucket existed working.
package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/ddmas26/inventory/internal/config"
)

// Store uploads objects to and builds public URLs for an S3-compatible bucket.
//
// A nil or disabled *Store is valid: every method either reports no-op state or
// returns the reference unchanged, which is what the local-disk fallback relies on.
type Store struct {
	client    *s3.Client
	bucket    string
	baseURL   string
	keyPrefix string
}

// New builds a Store from configuration. When no bucket is configured it returns
// a disabled Store and no error, so callers never need to special-case it.
func New(ctx context.Context, cfg config.StorageConfig) (*Store, error) {
	if !cfg.Enabled() {
		log.Println("storage: S3_BUCKET not set, product images will be stored on local disk")
		return &Store{}, nil
	}

	loadOpts := []func(*awsconfig.LoadOptions) error{
		// The SDK errors out when it cannot resolve a region, even against MinIO,
		// so the configured region is always applied.
		awsconfig.WithRegion(cfg.Region),
	}
	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		loadOpts = append(loadOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint == "" {
			return // real S3: standard regional endpoints, virtual-host style
		}
		// MinIO: explicit endpoint and path-style URLs (/bucket/key).
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = true
	})

	s := &Store{
		client:    client,
		bucket:    cfg.Bucket,
		baseURL:   strings.TrimRight(publicBaseURL(cfg), "/"),
		keyPrefix: strings.Trim(cfg.KeyPrefix, "/"),
	}

	if cfg.CreateBucket {
		if err := s.ensureBucket(ctx); err != nil {
			return nil, err
		}
	}

	log.Printf("storage: bucket %q ready (base URL %s)", s.bucket, s.baseURL)
	return s, nil
}

// publicBaseURL works out where images are served from.
func publicBaseURL(cfg config.StorageConfig) string {
	if cfg.PublicBaseURL != "" {
		return cfg.PublicBaseURL
	}
	if cfg.Endpoint != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(cfg.Endpoint, "/"), cfg.Bucket)
	}
	// Standard AWS virtual-host URL, e.g. https://my-bucket.s3.eu-central-1.amazonaws.com
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.Bucket, cfg.Region)
}

// Enabled reports whether objects are stored in a bucket.
func (s *Store) Enabled() bool {
	return s != nil && s.client != nil && s.bucket != ""
}

// BaseURL is where images in this bucket are served from.
func (s *Store) BaseURL() string {
	if s == nil {
		return ""
	}
	return s.baseURL
}

// ensureBucket creates the bucket if it is missing and makes it publicly
// readable so the browser can load images straight from MinIO. This only runs
// when CreateBucket is set, which defaults to true for a custom endpoint
// (MinIO) and false for real AWS.
func (s *Store) ensureBucket(ctx context.Context) error {
	if _, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)}); err == nil {
		return nil
	}

	log.Printf("storage: creating bucket %q", s.bucket)
	if _, err := s.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(s.bucket)}); err != nil {
		return fmt.Errorf("create bucket %q: %w", s.bucket, err)
	}

	return s.SetPublicReadPolicy(ctx)
}

// SetPublicReadPolicy allows anonymous GetObject on the bucket.
func (s *Store) SetPublicReadPolicy(ctx context.Context) error {
	policy := fmt.Sprintf(`{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {"AWS": ["*"]},
    "Action": ["s3:GetObject"],
    "Resource": ["arn:aws:s3:::%s/*"]
  }]
}`, s.bucket)

	if _, err := s.client.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
		Bucket: aws.String(s.bucket),
		Policy: aws.String(policy),
	}); err != nil {
		return fmt.Errorf("set public read policy on %q: %w", s.bucket, err)
	}
	return nil
}

// NewKey returns a fresh, unique object key for the given file extension, for
// example "products/9f2c1d….jpg".
func (s *Store) NewKey(ext string) string {
	name := RandomHex(16) + ext
	if s == nil || s.keyPrefix == "" {
		return name
	}
	return s.keyPrefix + "/" + name
}

// Upload stores an object under key.
func (s *Store) Upload(ctx context.Context, key string, body io.Reader, contentType string, size int64) error {
	if !s.Enabled() {
		return errors.New("storage: no bucket configured")
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	}
	if size > 0 {
		input.ContentLength = aws.Int64(size)
	}

	if _, err := s.client.PutObject(ctx, input); err != nil {
		return fmt.Errorf("upload %q: %w", key, err)
	}
	return nil
}

// Delete removes an object. Called for keys that may no longer exist.
func (s *Store) Delete(ctx context.Context, key string) error {
	if !s.Enabled() {
		return nil
	}

	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}); err != nil {
		return fmt.Errorf("delete %q: %w", key, err)
	}
	return nil
}

// Ref canonicalises a client-supplied value into what gets stored in the
// database. A URL pointing at our own bucket becomes an object key; external
// URLs and site-relative paths (legacy /uploads/… files) are kept as they are.
func (s *Store) Ref(raw string) string {
	ref := strings.TrimSpace(raw)
	if ref == "" || s == nil {
		return ref
	}

	if key, ok := s.keyFromURL(ref); ok {
		return key
	}
	return ref
}

// URL turns a stored reference into something the browser can load.
func (s *Store) URL(ref string) string {
	if ref == "" || s == nil {
		return ref
	}
	// External image, or a legacy file served from local disk.
	if isAbsoluteURL(ref) || strings.HasPrefix(ref, "/") {
		return ref
	}
	if s.baseURL == "" {
		return ref
	}
	return s.baseURL + "/" + strings.TrimLeft(ref, "/")
}

// KeyFromRef returns the object key for a stored reference, if it is one.
func (s *Store) KeyFromRef(ref string) (string, bool) {
	if s == nil || ref == "" || isAbsoluteURL(ref) || strings.HasPrefix(ref, "/") {
		return "", false
	}
	return ref, true
}

// keyFromURL extracts the object key when the URL points at our own bucket.
func (s *Store) keyFromURL(raw string) (string, bool) {
	if s == nil || s.baseURL == "" || !isAbsoluteURL(raw) {
		return "", false
	}

	prefix := s.baseURL + "/"
	if !strings.HasPrefix(raw, prefix) {
		return "", false
	}

	key := strings.TrimPrefix(raw, prefix)
	if key == "" {
		return "", false
	}
	return key, true
}

func isAbsoluteURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// RandomHex returns a cryptographically random hex string of n bytes.
func RandomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
