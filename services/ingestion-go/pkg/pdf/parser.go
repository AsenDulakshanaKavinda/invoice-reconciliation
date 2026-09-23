package pdf

// This package provides functionality to parse PDF files stored in a MinIO bucket and extract their text content.
// It uses the "github.com/ledongthuc/pdf" library to read and extract text from PDF files and the "github.com/minio/minio-go/v7" library to interact with MinIO.

import (
	"context"
	"io"

	"github.com/ledongthuc/pdf"
	"github.com/minio/minio-go/v7"
)

// ParsePDF extracts text from a PDF file stored in a MinIO bucket.
// It takes a context, a MinIO client, the bucket name, and the object name as parameters.
// It returns the extracted text as a string and an error if any occurred during the process.
func ParsePDF(ctx context.Context, client *minio.Client, bucket string, objectName string) (string, error) {

	// Get the object from MinIO
	object, err := client.GetObject(ctx, bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return "", err
	}
	defer object.Close()

	// Get the object statistics to determine its size
	stat, err := object.Stat()
	if err != nil {
		return "", nil
	}

	// Create a new PDF reader from the object
	reader, err := pdf.NewReader(object, stat.Size)
	if err != nil {
		return "", err
	}

	// Extract plain text from the PDF reader
	textReader, err := reader.GetPlainText()
	if err != nil {
		return "", err
	}

	// Read all the text from the text reader
	buf, err := io.ReadAll(textReader)
	if err != nil {
		return "", err
	}

	return string(buf), nil

}
