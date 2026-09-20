package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds everything the server needs. Every value can be overridden with
// an environment variable; the defaults match docker-compose.yml so that
// `go run .` works on a laptop with no extra setup.
type Config struct {
	Addr string

	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioRegion    string
	MinioUseSSL    bool

	DatabaseURL string

	PresignExpiry  time.Duration
	MaxUploadBytes int64

	Environment string
}

func LoadConfig() Config {
	return Config{
		Addr: getenv("ADDR", ":8080"),

		// This host ends up inside the presigned URL, so the *browser* must be
		// able to reach it. localhost:9000 works when MinIO runs via compose.
		MinioEndpoint:  getenv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey: getenv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey: getenv("MINIO_SECRET_KEY", "minioadminpassword"),
		MinioBucket:    getenv("MINIO_BUCKET", "my-sample-bucket"),
		MinioRegion:    getenv("MINIO_REGION", "us-east-1"),
		MinioUseSSL:    getenvBool("MINIO_USE_SSL", false),

		DatabaseURL: getenv("DATABASE_URL",
			"postgres://postgres:postgrespassword@localhost:5432/invoice_audit_db?sslmode=disable"),

		PresignExpiry:  getenvDuration("PRESIGN_EXPIRY", 15*time.Minute),
		MaxUploadBytes: getenvInt64("MAX_UPLOAD_BYTES", 20<<20), // 20 MiB

		Environment: getenv("ENV", "dev"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func getenvInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return fallback
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
