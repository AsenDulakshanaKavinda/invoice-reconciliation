package storage

import (
	"context"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/notification"
)

// BucketNotificationInput defines the parameters for setting up a bucket notification listener.
// It includes the bucket name, optional prefix and suffix filters, and the list of events to listen for.
type BucketNotificationInput struct {
	BucketName string
	Prefix string
	Suffix string
	Events []string
}

// WatchChangesInStorage sets up a listener for changes in the specified MinIO/S3 bucket based on the provided notification input.
// It returns a channel that emits notification.Info objects whenever a relevant event occurs in the bucket.
func WatchChangesInStorage(ctx context.Context, mc *minio.Client , bn *BucketNotificationInput) <-chan notification.Info{
	notificationInfoChain := mc.ListenBucketNotification(
		ctx, bn.BucketName, bn.Prefix, bn.Suffix, bn.Events,
	)

	log.Printf("Listening for notifications on bucket: %s...", bn.BucketName)

	return notificationInfoChain
}

