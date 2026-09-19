package handlers

import (
	"fmt"
	"mock-uploader/config"
	"path"
	"regexp"
	"strings"
	"time"

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



