package handlers

import (
	"errors"
	"fmt"
	"log"
	"mock-uploader/config"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
	"uuid"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
)

type Server struct {
	Cfg   config.Config
	DB    *pgxpool.Pool
	Minio *minio.Client
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
		name = fmt.Sprintf("%s_%s", space, time.Now().Format(time.RFC3339))
	}

	if len(name) > 120 {
		ext := path.Ext(name)
		base := strings.TrimSuffix(name, ext)
		if len(ext) < 120 {
			base = base[:120-len(ext)]
			name = base + ext
		} else {
			name = name[len(name)-120:]
		}
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

func (s *Server) CreateUpload(c *gin.Context) {
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
		fail(c, http.StatusUnsupportedMediaType, "only PDF files are accepted")
		return
	}

	// validate file size
	// if file size larger than max byte size return error (413)
	if req.Size > s.Cfg.MaxUploadBytes {
		fail(
			c,
			http.StatusRequestEntityTooLarge,
			fmt.Sprintf("file is larger than the %d MB limit", s.Cfg.MaxUploadBytes>>20),
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
	presigned, err := s.Minio.PresignedPutObject(ctx, s.Cfg.MinioBucket, key, s.Cfg.PresignExpiry)
	if err != nil {
		log.Printf("presign failed for %s: %v", key, err)
		fail(c, http.StatusInternalServerError, "could not generate upload URL")
		return
	}

	// insert invoice file details into the table
	_, err = s.DB.Exec(ctx,
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
		"expires_in_seconds": int(s.Cfg.PresignExpiry.Seconds()),
	})

}

// --- POST /api/uploads/:id/complete
func (s *Server) CompleteUpload(c *gin.Context) {

	// validate the id (it must ne a uuid)
	// if invalid return error (400)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid upload id")
		return
	}

	ctx := c.Request.Context()

	// check is there a invoice record with id
	// if it is read key and status
	// if not there is not record return error (404)
	// if found other type of error return (500) error
	var key, status string
	err = s.DB.QueryRow(ctx,
		`SELECT object_key, status FROM invoices WHERE id = $1`, id.String()).Scan(&key, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(c, http.StatusNotFound, "upload not found")
		return
	}
	if err != nil {
		log.Printf("lookup failed: %v", err)
		fail(c, http.StatusInternalServerError, "could not look up the upload")
		return
	}

	// if status of the record is uploaded - 200
	// if status of the record is pending - skip
	// else - 409
	switch status {
	case "uploaded":
		c.JSON(http.StatusOK, gin.H{"id": id.String(), "status": "uploaded"})
		return
	case "pending":
		// continue below
	default:
		fail(c, http.StatusConflict, "this upload was rejected; start a new one")
		return
	}

	// validate either record send to the s3 or not
	// if not return error (409)
	// if any other error (500)
	info, err := s.Minio.StatObject(ctx, s.Cfg.MinioBucket, key, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			fail(c, http.StatusConflict, "the file has not reached storage yet")
			return
		}
		log.Printf("stat failed for %s: %v", key, err)
		fail(c, http.StatusInternalServerError, "could not verify the uploaded file")
		return
	}

	// if record size > max upload size remove it
	// if there is a error while removing to log the error
	// and then remove it from the invoice record table
	// and return error (413)
	if info.Size > s.Cfg.MaxUploadBytes {
		if rmErr := s.Minio.RemoveObject(ctx, s.Cfg.MinioBucket, key, minio.RemoveObjectOptions{}); rmErr != nil {
			log.Printf("remove oversized object %s failed: %v", key, rmErr)
		}
		_, _ = s.DB.Exec(ctx, `UPDATE invoices SET status = 'failed' WHERE id = $1`, id.String())
		fail(c, http.StatusRequestEntityTooLarge, "the uploaded file exceeds the size limit and was removed")
		return
	}

	// set status -> uploaded and update time
	// if error return error (500)
	_, err = s.DB.Exec(ctx,
		`UPDATE invoices SET status = 'uploaded', size_bytes = $2, uploaded_at = now() WHERE id = $1`,
		id.String(), info.Size)
	if err != nil {
		log.Printf("update failed: %v", err)
		fail(c, http.StatusInternalServerError, "could not save the upload")
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id.String(), "status": "uploaded", "size_bytes": info.Size})

}

// GET /api/invoices
type invoiceItem struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	Status      string    `json:"status"`
	SizeBytes   int64     `json:"size_bytes"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *Server) ListInvoices(c *gin.Context) {

	// get last 20 invoice record upload
	// if there is a error return (500)
	rows, err := s.DB.Query(c.Request.Context(),
		`SELECT id::text, original_filename, content_type, status, COALESCE(size_bytes, 0), created_at
		 FROM invoices
		 ORDER BY created_at DESC
		 LIMIT 20`)
	if err != nil {
		log.Printf("list failed: %v", err)
		fail(c, http.StatusInternalServerError, "could not load invoices")
		return
	}
	defer rows.Close()

	// store insight of invoice items in a array
	// if any error return (500) else return json obj (200, items)
	items := []invoiceItem{}
	for rows.Next() {
		var it invoiceItem
		if err := rows.Scan(&it.ID, &it.Filename, &it.ContentType, &it.Status, &it.SizeBytes, &it.CreatedAt); err != nil {
			log.Printf("scan failed: %v", err)
			fail(c, http.StatusInternalServerError, "could not load invoices")
			return
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		log.Printf("rows failed: %v", err)
		fail(c, http.StatusInternalServerError, "could not load invoices")
		return
	}
	c.JSON(http.StatusOK, items)

}

// --- GET /api/invoices/:id/download
func (s *Server) DownloadInvoice(c *gin.Context) {
	// parse and validate id
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid invoice id")
		return
	}

	ctx := c.Request.Context()

	// fetch record, get key, filename and status of the item
	// if record not found return error (404)
	// if the status is not uploaded return (409)
	// for other errors return error (500)
	var key, filename, status string
	err = s.DB.QueryRow(ctx,
		`SELECT object_key, original_filename, status FROM invoices WHERE id = $1`,
		id.String()).Scan(&key, &filename, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(c, http.StatusNotFound, "invoice not found")
		return
	}
	if err != nil {
		log.Printf("lookup failed: %v", err)
		fail(c, http.StatusInternalServerError, "could not look up the invoice")
		return
	}
	if status != "uploaded" {
		fail(c, http.StatusConflict, "this invoice has no stored file")
		return
	}

	// creates a temporary link that lets the browser download a stored file
	// if error return (500)
	// else send json obj (200, download url)
	params := url.Values{}
	params.Set("response-content-disposition",
		fmt.Sprintf(`attachment; filename="%s"`, sanitizeFilename(filename)))

	link, err := s.Minio.PresignedGetObject(ctx, s.Cfg.MinioBucket, key, 5*time.Minute, params)
	if err != nil {
		log.Printf("presign get failed for %s: %v", key, err)
		fail(c, http.StatusInternalServerError, "could not generate download URL")
		return
	}
	c.JSON(http.StatusOK, gin.H{"download_url": link.String()})

}
