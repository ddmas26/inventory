package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the application.
type Config struct {
	DB        DatabaseConfig
	Redis     RedisConfig
	Auth      AuthConfig
	Storage   StorageConfig
	UploadDir string // fallback directory for product images when S3 is not configured
}

// StorageConfig describes the S3-compatible object store used for product images.
//
// The same code talks to MinIO in development and AWS S3 in production: point
// Endpoint at MinIO (which also switches on path-style URLs) and leave it empty
// for real S3, where the SDK uses the standard regional endpoints.
type StorageConfig struct {
	// Bucket holds the images. When empty, S3 is disabled and uploads fall back
	// to the local UploadDir.
	Bucket string
	// Region is required by the AWS SDK. MinIO ignores the value but the SDK
	// still refuses to start without one.
	Region string
	// Endpoint is the custom S3 endpoint, e.g. "http://localhost:9000" for MinIO.
	// Leave empty for AWS S3.
	Endpoint string
	// PublicBaseURL is the base URL images are served from. When empty it is
	// derived: "{Endpoint}/{Bucket}" for MinIO, or the standard AWS virtual-host
	// URL for S3.
	PublicBaseURL string
	// AccessKey and SecretKey are only needed when the SDK's default credential
	// chain (AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY, shared config, IAM role)
	// is not used.
	AccessKey string
	SecretKey string
	// SessionToken accompanies temporary credentials (AWS STS, SSO, IAM Identity
	// Center, AWS Academy). Omitting it for temporary credentials makes S3 reject
	// every request with SignatureDoesNotMatch.
	SessionToken string
	// KeyPrefix namespaces uploaded objects inside the bucket.
	KeyPrefix string
	// CreateBucket creates the bucket (and a public read policy) on startup when
	// it is missing. Intended for local development only.
	CreateBucket bool
}

// AuthConfig holds the lifetimes used by the Redis-backed token store.
type AuthConfig struct {
	// AccessTokenTTL is how long an access token (session) stays valid.
	AccessTokenTTL time.Duration
	// RefreshTokenTTL is how long a refresh token can be exchanged for a new pair.
	RefreshTokenTTL time.Duration
}

// DatabaseConfig holds PostgreSQL connection parameters.
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// DSN returns the PostgreSQL connection string.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

// RedisConfig holds Redis connection parameters.
type RedisConfig struct {
	Host     string
	Port     string
	Password string
}

// Addr returns the Redis address string.
func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		DB: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "inventory"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
		},
		Auth: AuthConfig{
			AccessTokenTTL:  getEnvDuration("ACCESS_TOKEN_TTL", 2*time.Hour),
			RefreshTokenTTL: getEnvDuration("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		},
		Storage:   loadStorageConfig(),
		UploadDir: getEnv("UPLOAD_DIR", "uploads"),
	}
}

// loadStorageConfig reads S3/MinIO settings from the environment.
func loadStorageConfig() StorageConfig {
	endpoint := getEnv("S3_ENDPOINT", "")

	return StorageConfig{
		Bucket:        strings.TrimSpace(getEnv("S3_BUCKET", "")),
		Region:        strings.TrimSpace(getEnv("AWS_REGION", "us-east-1")),
		Endpoint:      strings.TrimSpace(endpoint),
		PublicBaseURL: strings.TrimSpace(getEnv("S3_PUBLIC_BASE_URL", "")),
		// Trailing spaces/newlines copied along with credentials are a very common
		// cause of SignatureDoesNotMatch, so every credential is trimmed here.
		AccessKey:    strings.TrimSpace(getEnv("AWS_ACCESS_KEY_ID", "")),
		SecretKey:    strings.TrimSpace(getEnv("AWS_SECRET_ACCESS_KEY", "")),
		SessionToken: strings.TrimSpace(getEnv("AWS_SESSION_TOKEN", "")),
		KeyPrefix:    getEnv("S3_KEY_PREFIX", "products"),
		// Only manage the bucket automatically against a custom endpoint (MinIO),
		// never against real AWS where the bucket and its policy are managed
		// outside the application.
		CreateBucket: getEnvBool("S3_CREATE_BUCKET", endpoint != ""),
	}
}

// Enabled reports whether an S3-compatible bucket is configured.
func (s StorageConfig) Enabled() bool {
	return s.Bucket != ""
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

// getEnvDuration reads a Go duration string (e.g. "30m", "168h") from the
// environment, falling back to the default when unset or invalid.
func getEnvDuration(key string, fallback time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	d, err := time.ParseDuration(val)
	if err != nil || d <= 0 {
		log.Printf("config: invalid %s=%q, falling back to %s", key, val, fallback)
		return fallback
	}
	return d
}

// getEnvBool reads a boolean from the environment, falling back when unset or
// unparseable.
func getEnvBool(key string, fallback bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		log.Printf("config: invalid %s=%q, falling back to %t", key, val, fallback)
		return fallback
	}
	return b
}
