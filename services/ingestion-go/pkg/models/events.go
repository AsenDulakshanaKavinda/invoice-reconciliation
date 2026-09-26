package models

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
	Content    string
}
