package handlers

import (
	"bytes"
	"mock-uploader/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Setup test Gin engine
func setupRouter(s *Server) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/api/uploads", s.CreateUpload)
	r.POST("/api/uploads/:id/complete", s.CompleteUpload)
	r.GET("/api/invoices", s.ListInvoices)
	r.GET("/api/invoices/:id/download", s.DownloadInvoice)

	return r
}

// --- 1. Unit Test for Helper Function ---
func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean simple name",
			input:    "document.pdf",
			expected: "document.pdf",
		},
		{
			name:     "path traversal unix",
			input:    "../../etc/passwd",
			expected: "passwd",
		},
		{
			name:     "path traversal windows",
			input:    `C:\Users\Secret\file.pdf`,
			expected: "file.pdf",
		},
		{
			name:     "unsafe symbols replaced",
			input:    "my invoice #1 (@2026).pdf",
			expected: "my_invoice_1_2026_.pdf",
		},
		{
			name:     "truncate long filename over 120 chars",
			input:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.pdf", // 125 chars total (121 'a's + 4 char ".pdf")
			expected: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.pdf",          // 120 chars total (116 'a's + 4 char ".pdf")
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// --- 2. HTTP Validation Tests (No DB/MinIO needed) ---
func TestCreateUpload_ValidationErrors(t *testing.T) {
	server := &Server{
		Cfg: config.Config{
			MaxUploadBytes: 10 * 1024 * 1024, // 10MB
		},
	}
	router := setupRouter(server)

	tests := []struct {
		name           string
		payload        string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "missing required size field",
			payload:        `{"filename": "test.pdf", "content_type": "application/pdf"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "filename, content_type and a non-zero size are required",
		},
		{
			name:           "unsupported media type",
			payload:        `{"filename": "test.png", "content_type": "image/png", "size": 1024}`,
			expectedStatus: http.StatusUnsupportedMediaType,
			expectedError:  "only PDF files are accepted",
		},
		{
			name:           "file size exceeds limit",
			payload:        `{"filename": "test.pdf", "content_type": "application/pdf", "size": 20000000}`,
			expectedStatus: http.StatusRequestEntityTooLarge,
			expectedError:  "file is larger than the 10 MB limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/uploads", bytes.NewBufferString(tt.payload))
			req.Header.Set("Content-Type", "application/json")

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedError)
		})
	}
}
