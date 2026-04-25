package web

import (
	"embed"
	"io/fs"
)

//go:embed static/*
var staticFiles embed.FS

func Files() fs.FS {
	files, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	return files
}
