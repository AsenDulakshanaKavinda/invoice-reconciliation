// Package main serves as the entry point for the mock-uploader application.
// It initializes core infrastructure components—including database connections,
// database schema migrations, and MinIO object storage—and sets up a Gin HTTP router
// with graceful shutdown capabilities.

// start server | invoice-reconciliation/services/mock-uploader>  go run ./cmd

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mockuploader "mock-uploader"
	"mock-uploader/config"
	"mock-uploader/handlers"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pressly/goose/v3"
)

// startupAttempts defines the maximum number of retry attempts for connecting
// to dependent services (PostgreSQL, minIO) during application startup.
const startupAttempts = 20

func main() {
	// --- 1. configuration and context setup ---

	// load application configuration from env variables
	cfg := config.LoadConfig()

	// set up a parent context that automatically cancels when an OS interruption
	// signal (SIGINT/Ctrl+C, SIGTERM) is received to initiate graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- 2. database connection and migration (dev only) ---

	// Establish a connection pool to PostgreSQL with retry mechanism
	pool, err := connectDB(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	if cfg.Environment == "dev" {
		// Execute pending database schema migrations using Goose.
		// TODO: Move migration execution to CI/CD or dedicated deployment pipelines.
		if err := runMigrations(ctx, pool); err != nil {
			log.Fatalf("migrate: %v", err)
		}
	}

	// --- 3. MinIO / S3 object storage client setup --

	// initialize the MinIO API client with configured credentials and endpoint settings
	mc, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioUseSSL,
		Region: cfg.MinioRegion,
	})
	if err != nil {
		log.Fatalf("minio client: %v", err)
	}

	// ensure the designated S3/MinIO bucket exists; create it if missing
	if err := ensureBucket(ctx, mc, cfg.MinioBucket, cfg.MinioRegion); err != nil {
		log.Fatalf("minio bucket: %v", err)
	}

	// initialize application route handlers with shared dependencies.
	srv := &handlers.Server{Cfg: cfg, DB: pool, Minio: mc}

	// --- 4. HTTP Router & Route Definitions ---

	router := gin.Default()

	// Root route: Serves the web UI index HTML
	router.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", mockuploader.IndexHTML)
	})

	// health check endpoint for liveness/readiness probes
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// REST API v1 endpoint group
	api := router.Group("/api")
	api.POST("/uploads", srv.CreateUpload)
	api.POST("/uploads/:id/complete", srv.CompleteUpload)
	api.GET("/invoices", srv.ListInvoices)
	api.GET("/invoices/:id/download", srv.DownloadInvoice)

	// --- 5. HTTP server execution and graceful shutdown ---

	httpSrv := &http.Server{Addr: cfg.Addr, Handler: router}

	// background worker routine to handle graceful server shutdown upon context cancellation
	go func() {
		<-ctx.Done() // block until shutdown signal (SIGINT/SIGTERM) is received

		// grant active HTTP requests a 5-second deadline to complete before forcing termination.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			log.Printf("http server shutdown error: %v", err)
		}
	}()

	log.Printf("listening on %s (open http://localhost%s)", cfg.Addr, cfg.Addr)

	// start listening for incoming network connections.
	// http.ErrServerClosed is expected when Shutdown is invoked.
	if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server: %v", err)
	}
}

// connectDB establishes a pgx pool connection to PostgreSQL, retrying up to startupAttempts
// times to handle transient connection delays (e.g., waiting for database container startup)
func connectDB(ctx context.Context, url string) (*pgxpool.Pool, error) {
	var lastErr error
	for i := 1; i <= startupAttempts; i++ {
		pool, err := pgxpool.New(ctx, url)
		if err == nil {
			if err = pool.Ping(ctx); err == nil {
				return pool, nil
			}
			pool.Close()
		}
		lastErr = err
		log.Printf("waiting for postgres (%d/%d): %v", i, startupAttempts, err)
		if err := sleep(ctx, time.Second); err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("postgres not reachable: %w", lastErr)
}

// runMigrations executes embedded/file SQL migrations using Goose via pgx stdlib adapter
// register pgx driver with database/sql for Goose compatibility.
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// register pgx driver with database/sql for Goose compatibility.
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}

	// run up migrations located in the default migrations directory.
	if err := goose.UpContext(ctx, db, "migrations"); err != nil {
		return fmt.Errorf("up migrations: %w", err)
	}

	return nil
}

// ensureBucket checks whether the specified MinIO/S3 bucket exists and creates it if missing
func ensureBucket(ctx context.Context, mc *minio.Client, bucket string, region string) error {
	var lastErr error
	for i := 1; i <= startupAttempts; i++ {
		exists, err := mc.BucketExists(ctx, bucket)
		if err == nil {
			if exists {
				return nil
			}
			err = mc.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: region})
			if err == nil {
				log.Printf("created bucket %q", bucket)
				return nil
			}
		}
		lastErr = err
		log.Printf("waiting for minio (%d/%d): %v", i, startupAttempts, err)
		if err := sleep(ctx, time.Second); err != nil {
			return err
		}
	}
	return fmt.Errorf("minio not reachable: %w", lastErr)
}

func sleep(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
