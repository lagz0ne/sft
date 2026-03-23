package web

import (
	"embed"
	"io/fs"
)

//go:embed all:apps/web/dist/client
var dist embed.FS

// ClientFS is the SPA root (contains _shell.html + assets/).
var ClientFS, _ = fs.Sub(dist, "apps/web/dist/client")
