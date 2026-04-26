package web

import (
	"embed"
	"io/fs"
)

//go:embed static/*
var staticFiles embed.FS

func Files() (fs.FS, error) {
	return fs.Sub(staticFiles, "static")
}
