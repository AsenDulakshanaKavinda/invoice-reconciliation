package storage

import (
	"context"
	"ingestion-go/pkg/models"
	"ingestion-go/pkg/pdf"
	"ingestion-go/pkg/publisher"

	"log"
	"net/url"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/notification"
)

// WatchChangesInStorage sets up a listener for changes in the specified MinIO/S3 bucket based on the provided notification input.
// It returns a channel that emits notification.Info objects whenever a relevant event occurs in the bucket.
func WatchChangesInStorage(ctx context.Context, mc *minio.Client, bn *models.BucketNotificationInput) <-chan notification.Info {
	log.Printf("Listening for notifications on bucket: %s...", bn.BucketName)
	return mc.ListenBucketNotification(ctx, bn.BucketName, bn.Prefix, bn.Suffix, bn.Events)

}

// StreamNotificationsToRabbitMQ listens for storage notifications and processes them using the provided handler function.
// It takes a context, a MinIO client, a BucketNotificationInput struct, and a handler function as parameters.
// The handler function is called with a BucketNotificationInfo struct for each relevant event received.
func StreamNotificationsToRabbitMQ(
	ctx context.Context,
	mc *minio.Client,
	input *models.BucketNotificationInput,
	publisher *publisher.RabbitPublisher,
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

				content, err := pdf.ParsePDF(ctx, mc, input.BucketName, key)
				if err != nil {
					log.Fatalf("Error while parsing the: %s", key)
					content = ""
				}


				info := models.BucketNotificationInfo{
					EventName:  record.EventName,
					BucketName: record.S3.Bucket.Name,
					ObjectKey:  key,
					ObjectSize: record.S3.Object.Size,
					ETag:       record.S3.Object.ETag,
					EventTime:  record.EventTime,
					Content: content,
				}

				log.Printf("ready to publish: %s, from: %s", info.ObjectKey, info.BucketName)

				if err := publisher.PublishNotification(ctx, info); err != nil {
					log.Printf("Failed to push event for %s to RabbitMQ: %v", info.ObjectKey, err)
				} else {
					log.Printf("Successfully published [%s] event for file '%s' to queue '%s'",
						info.EventName, info.ObjectKey, publisher.Queue.Name)
				}
			}

		}
	}
}


