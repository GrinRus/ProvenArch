package api

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed ui_dist/**
var embeddedUIDist embed.FS

var uiFS = mustUISubFS()

func mustUISubFS() fs.FS {
	sub, err := fs.Sub(embeddedUIDist, "ui_dist")
	if err != nil {
		return nil
	}
	return sub
}

func EmbeddedUIAvailable() bool {
	if uiFS == nil {
		return false
	}
	if _, err := fs.Stat(uiFS, "index.html"); err != nil {
		return false
	}
	return true
}

func serveEmbeddedUI(writer http.ResponseWriter, request *http.Request) {
	if uiFS == nil {
		http.NotFound(writer, request)
		return
	}

	path := strings.TrimPrefix(request.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}
	if strings.HasSuffix(path, "/") {
		path += "index.html"
	}

	if _, err := fs.Stat(uiFS, path); err == nil {
		http.FileServer(http.FS(uiFS)).ServeHTTP(writer, request)
		return
	}

	indexContent, err := fs.ReadFile(uiFS, "index.html")
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(indexContent)
}
