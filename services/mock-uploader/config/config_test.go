package config

import (
	"testing"
	"time"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Ensure environment variables from the outside do not affect
	// this test.
	t.Setenv("ADDR", "")
	t.Setenv("MINIO_ENDPOINT", "")
	t.Setenv("MINIO_ACCESS_KEY", "")
	t.Setenv("MINIO_SECRET_KEY", "")
	t.Setenv("MINIO_BUCKET", "")
	t.Setenv("MINIO_REGION", "")
	t.Setenv("MINIO_USE_SSL", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PRESIGN_EXPIRY", "")
	t.Setenv("MAX_UPLOAD_BYTES", "")
	t.Setenv("ENV", "")

	cfg := LoadConfig()

	if cfg.Addr != ":8080" {
		t.Errorf("expected default Addr :8080, got %s", cfg.Addr)
	}

	if cfg.MinioEndpoint != "localhost:9000" {
		t.Errorf(
			"expected default MinioEndpoint localhost:9000, got %s",
			cfg.MinioEndpoint,
		)
	}

	if cfg.MinioAccessKey != "minioadmin" {
		t.Errorf(
			"expected default MinioAccessKey minioadmin, got %s",
			cfg.MinioAccessKey,
		)
	}

	if cfg.MinioSecretKey != "minioadminpassword" {
		t.Errorf(
			"expected default MinioSecretKey minioadminpassword, got %s",
			cfg.MinioSecretKey,
		)
	}

	if cfg.MinioBucket != "my-sample-bucket" {
		t.Errorf(
			"expected default MinioBucket my-sample-bucket, got %s",
			cfg.MinioBucket,
		)
	}

	if cfg.MinioRegion != "us-east-1" {
		t.Errorf(
			"expected default MinioRegion us-east-1, got %s",
			cfg.MinioRegion,
		)
	}

	if cfg.MinioUseSSL != false {
		t.Errorf(
			"expected default MinioUseSSL false, got %v",
			cfg.MinioUseSSL,
		)
	}

	expectedDBURL :=
		"postgres://postgres:postgrespassword@localhost:5432/invoice_audit_db?sslmode=disable"

	if cfg.DatabaseURL != expectedDBURL {
		t.Errorf(
			"expected default DatabaseURL %s, got %s",
			expectedDBURL,
			cfg.DatabaseURL,
		)
	}

	if cfg.PresignExpiry != 15*time.Minute {
		t.Errorf(
			"expected default PresignExpiry 15m, got %s",
			cfg.PresignExpiry,
		)
	}

	if cfg.MaxUploadBytes != 20<<20 {
		t.Errorf(
			"expected default MaxUploadBytes %d, got %d",
			int64(20<<20),
			cfg.MaxUploadBytes,
		)
	}
}

func TestLoadConfig_CustomEnvVars(t *testing.T) {
	t.Setenv("ADDR", ":9090")
	t.Setenv("MINIO_ENDPOINT", "minio:9000")
	t.Setenv("MINIO_ACCESS_KEY", "custom-access-key")
	t.Setenv("MINIO_SECRET_KEY", "custom-secret-key")
	t.Setenv("MINIO_BUCKET", "custom-bucket")
	t.Setenv("MINIO_REGION", "eu-west-1")
	t.Setenv("MINIO_USE_SSL", "true")
	t.Setenv(
		"DATABASE_URL",
		"postgres://user:password@db:5432/testdb?sslmode=disable",
	)
	t.Setenv("PRESIGN_EXPIRY", "30m")
	t.Setenv("MAX_UPLOAD_BYTES", "52428800")

	cfg := LoadConfig()

	if cfg.Addr != ":9090" {
		t.Errorf("expected Addr :9090, got %s", cfg.Addr)
	}

	if cfg.MinioEndpoint != "minio:9000" {
		t.Errorf(
			"expected MinioEndpoint minio:9000, got %s",
			cfg.MinioEndpoint,
		)
	}

	if cfg.MinioAccessKey != "custom-access-key" {
		t.Errorf(
			"expected MinioAccessKey custom-access-key, got %s",
			cfg.MinioAccessKey,
		)
	}

	if cfg.MinioSecretKey != "custom-secret-key" {
		t.Errorf(
			"expected MinioSecretKey custom-secret-key, got %s",
			cfg.MinioSecretKey,
		)
	}

	if cfg.MinioBucket != "custom-bucket" {
		t.Errorf(
			"expected MinioBucket custom-bucket, got %s",
			cfg.MinioBucket,
		)
	}

	if cfg.MinioRegion != "eu-west-1" {
		t.Errorf(
			"expected MinioRegion eu-west-1, got %s",
			cfg.MinioRegion,
		)
	}

	if !cfg.MinioUseSSL {
		t.Errorf("expected MinioUseSSL true, got false")
	}

	expectedDBURL :=
		"postgres://user:password@db:5432/testdb?sslmode=disable"

	if cfg.DatabaseURL != expectedDBURL {
		t.Errorf(
			"expected DatabaseURL %s, got %s",
			expectedDBURL,
			cfg.DatabaseURL,
		)
	}

	if cfg.PresignExpiry != 30*time.Minute {
		t.Errorf(
			"expected PresignExpiry 30m, got %s",
			cfg.PresignExpiry,
		)
	}

	if cfg.MaxUploadBytes != 52428800 {
		t.Errorf(
			"expected MaxUploadBytes 52428800, got %d",
			cfg.MaxUploadBytes,
		)
	}
}

/*

### Run the tests

From the project root:
go test ./...


For just the config package:
go test ./config


And for more detailed output:
go test -v ./config

*/