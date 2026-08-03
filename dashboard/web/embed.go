package web

import (
	"embed"
	"io/fs"
)

// embedded contains the production frontend build.
//
//go:embed dist/*
var embedded embed.FS

func Dist() (fs.FS, error) {
	return fs.Sub(embedded, "dist")
}
