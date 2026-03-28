package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
)

//go:embed dist dist/*
var distFS embed.FS

func Handler() http.Handler {
	dist, err := fs.Sub(distFS, "dist")
	if err != nil {
		return http.NotFoundHandler()
	}

	fileServer := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		f, openErr := dist.Open(path)
		if openErr != nil {
			if isAssetPath(path) || (filepath.Ext(filepath.Base(path)) != "" && !wantsHTMLNavigation(r)) {
				http.NotFound(w, r)
				return
			}
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}
		_ = f.Close()

		fileServer.ServeHTTP(w, r)
	})
}

var assetPrefixes = []string{"assets/", "static/", "js/", "css/", "img/", "fonts/", "images/"}

func isAssetPath(path string) bool {
	for _, prefix := range assetPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func wantsHTMLNavigation(r *http.Request) bool {
	accept := strings.ToLower(strings.TrimSpace(r.Header.Get("Accept")))
	if accept == "" {
		return false
	}
	return strings.Contains(accept, "text/html")
}
