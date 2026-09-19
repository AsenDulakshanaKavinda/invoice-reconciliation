package handlers

import (

	"fmt"
	"log"
	"mock-uploader/config"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"
	"uuid"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
)


type Server struct {
	cfg config.Config
	db *pgxpool.Pool
	minio *minio.Client
}

var allowedContentTypes = map[string]bool{
	"application/pdf": true,
}

var unsafeFilenameChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// sanitizeFilename keeps object keys predictable: no path separators, no
// spaces, no exotic characters, bounded length.
func sanitizeFilename(name string) string {
	name = path.Base(strings.ReplaceAll(name, "\\", "/"))
	name = unsafeFilenameChars.ReplaceAllString(name, "_")
	name = strings.Trim(name, "._")
	if name == "" {
		space := "missing_filename"
		name = fmt.Sprintf("%s: %s", space, time.Now().Format(time.RFC3339))
	}
	if len(name) > 120 {
		name = name[len(name)-120:] // keep the extension
	}
	return name
}

func fail(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"error": msg})
}

// --- POST - /api/uploads
type createUploadRequest struct {
	Filename    string `json:"filename" binding:"required"`
	ContentType string `json:"content_type" binding:"required"`
	Size        int64  `json:"size" binding:"required,gt=0"`
}

func (s *Server) createUpload(c *gin.Context) {
	var req createUploadRequest

	// Deserialize - validate and return Error
	// read the HTTP req body, parses the JSON data and 
	// assing it to struct and return error (400)
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "filename, content_type and a non-zero size are required")
		return
	}

	// validate file content type
	// if file content type not in allowedContentTypes map
	// return error (415)
	if !allowedContentTypes[req.ContentType] {
		fail(c, http.StatusUnsupportedMediaType, "only PDF, PNG and JPEG files are accepted")
		return
	}

	// validate file size
	// if file size larger than max byte size return error (413)
	if req.Size > s.cfg.MaxUploadBytes {
		fail(
			c,
			http.StatusRequestEntityTooLarge,
			fmt.Sprintf("file is larger than the %d MB limit", s.cfg.MaxUploadBytes>>20),
		)
		return
	}

	ctx := c.Request.Context()
	id := uuid.New()
	// The UUID prefix means two people uploading "invoice.pdf" never collide.
	key := fmt.Sprintf("invoices/%s/%s", id, sanitizeFilename(req.Filename))

	// Create a presigned PUT URL: a temporary, signed link that lets the browser
	// upload exactly one object (this bucket + key) straight to MinIO, for
	// PresignExpiry (15 min). The secret key is only used to sign it and is never
	// included in the URL. This is computed locally (the region is set on the
	// client), so it makes no network call to MinIO.
	presigned, err := s.minio.PresignedPutObject(ctx, s.cfg.MinioBucket, key, s.cfg.PresignExpiry)
	if err != nil {
		log.Printf("presign failed for %s: %v", key, err)
		fail(c, http.StatusInternalServerError, "could not generate upload URL")
		return
	}

	// insert invoice file details into the table
	_, err = s.db.Exec(ctx,
		`INSERT INTO invoices (id, object_key, original_filename, content_type, declared_size)
		 VALUES ($1, $2, $3, $4, $5)`,
		id.String(), key, req.Filename, req.ContentType, req.Size)

	if err != nil {
		log.Printf("insert invoice failed: %v", err)
		fail(c, http.StatusInternalServerError, "could not record the upload")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                 id.String(),
		"object_key":         key,
		"upload_url":         presigned.String(),
		"content_type":       req.ContentType,
		"expires_in_seconds": int(s.cfg.PresignExpiry.Seconds()),
	})

}






