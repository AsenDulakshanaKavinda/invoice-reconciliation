package storage

// This file contains utility functions for interacting with MinIO/S3 storage, including client creation and bucket validation.
// It provides a convenient way to set up and verify storage configurations for the ingestion service.


import (
	"context"
	"ingestion-go/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)


// create a new minio client using the provided configuration
// returns the client and any error encountered during creation
func CreateMinIOClient(cfg config.Config) (*minio.Client, error) {
	mc, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds: credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
	})

	if err != nil {
		return nil, err
	}
	return mc, nil
}


// validateBucket checks if the specified bucket exists in the MinIO/S3 storage.
// returns true if the bucket exists, false otherwise, along with any error encountered.
func ValidateBucket(ctx context.Context, mc *minio.Client, bucket string) (bool, error) {
	exists, err := mc.BucketExists(
		ctx, bucket,
	)

	if err != nil {
		return false, err
	}
	return exists, nil
}