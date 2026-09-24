package main

import (
	"context"
	"ingestion-go/config"
	"ingestion-go/pkg/models"
	"ingestion-go/pkg/publisher"
	"ingestion-go/pkg/storage"
	"log"
	"os"
	"os/signal"
	"syscall"
)


func main() {
	cfg := config.LoadConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. initialize MinIO client
	minioClient, err := storage.CreateMinIOClient(cfg)
	if err != nil {
		log.Fatalf("Error while initializing MinIO client, %s", err)
		return
	}

	// 2. initialize RabbitMQ publisher
	rp, err := publisher.NewRabbitPublisher(cfg)
	if err != nil {
		log.Fatalf("Error while initializing RabbitMQ publisher, %s", err)
	}
	defer rp.Close()

	// 3. start streaming notifications from MinIO to RabbitMQ
	input := &models.BucketNotificationInput{
		BucketName: cfg.MinioBucket,
		Prefix: "",
		Suffix: "",
		Events: []string{"s3:ObjectCreated:*", "s3:ObjectRemoved:*"},
	}

	


	// 4. Start streaming notifications in a separate goroutine
	go storage.StreamNotificationsToRabbitMQ(ctx, minioClient, input, rp)

	// 5. Wait for termination signal (e.g., Ctrl+C) to gracefully shut down
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")


}