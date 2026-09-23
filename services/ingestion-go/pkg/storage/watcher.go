package storage

import (
	"context"
	"log"
	"net/url"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/notification"
)

// BucketNotificationInput defines the parameters for setting up a bucket notification listener.
// It includes the bucket name, optional prefix and suffix filters, and the list of events to listen for.

// BucketNotificationInfo represents the information extracted from a bucket notification event.
type BucketNotificationInput struct {
	BucketName string
	Prefix     string
	Suffix     string
	Events     []string
}

// WatchChangesInStorage sets up a listener for changes in the specified MinIO/S3 bucket based on the provided notification input.
type BucketNotificationInfo struct {
	EventName  string
	BucketName string
	ObjectKey  string
	ObjectSize int64
	ETag       string
	EventTime  string
}

// WatchChangesInStorage sets up a listener for changes in the specified MinIO/S3 bucket based on the provided notification input.
// It returns a channel that emits notification.Info objects whenever a relevant event occurs in the bucket.
func WatchChangesInStorage(ctx context.Context, mc *minio.Client, bn *BucketNotificationInput) <-chan notification.Info {
	log.Printf("Listening for notifications on bucket: %s...", bn.BucketName)
	return mc.ListenBucketNotification(ctx, bn.BucketName, bn.Prefix, bn.Suffix, bn.Events)

}

// HandleStorageNotifications listens for storage notifications and processes them using the provided handler function.
// It takes a context, a MinIO client, a BucketNotificationInput struct, and a handler function as parameters.
// The handler function is called with a BucketNotificationInfo struct for each relevant event received.
func HandleStorageNotifications(
	ctx context.Context,
	mc *minio.Client,
	input *BucketNotificationInput,
	handler func(info BucketNotificationInfo),
) {
	// Start listening for notifications in a separate goroutine
	notificationChan := WatchChangesInStorage(ctx, mc, input)

	for {
		select {
		// Exit the loop if the context is canceled
		case <-ctx.Done():
			log.Println("Stopping notification listener context closed")
			return
		// Receive notification from the channel
		case notificationInfo, ok := <-notificationChan:
			// If the channel is closed, log and exit the loop
			if !ok {
				log.Println("Notification channel closed")
				return
			}

			// If there's an error in the notification, log it and continue to the next iteration
			if notificationInfo.Err != nil {
				log.Printf("Error receiving notification: %v", notificationInfo.Err)
				continue
			}

			// Process each record in the notification
			for _, record := range notificationInfo.Records {
				key, err := url.QueryUnescape(record.S3.Object.Key)
				if err != nil {
					key = record.S3.Object.Key
				}

				info := BucketNotificationInfo{
					EventName:  record.EventName,
					BucketName: record.S3.Bucket.Name,
					ObjectKey:  key,
					ObjectSize: record.S3.Object.Size,
					ETag:       record.S3.Object.ETag,
					EventTime:  record.EventTime,
				}

				if handler != nil {
					handler(info)
				}
			}

		}
	}
}

// use -
/*
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    input := &storage.BucketNotificationInput{
        BucketName: "documents",
        Events:     []string{"s3:ObjectCreated:*", "s3:ObjectRemoved:*"},
    }

    // Process each notification stream event dynamically
    go storage.HandleStorageNotifications(ctx, minioClient, input, func(info storage.BucketNotificationInfo) {
        log.Printf("Received Event [%s] for File: %s (%d bytes)", info.EventName, info.ObjectKey, info.ObjectSize)
    })

    // Keep service running...
    select {}
}

*/
