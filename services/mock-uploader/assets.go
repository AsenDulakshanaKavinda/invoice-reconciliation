package mockuploader

import "embed"

//go:embed web/index.html
var IndexHTML []byte

//go:embed migrations/*.sql
var Migrations embed.FS