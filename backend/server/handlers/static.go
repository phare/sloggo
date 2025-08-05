package handlers

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// StaticHandler serves static files from the frontend build
// It also handles the SPA routing by serving index.html for routes that don't exist
func StaticHandler(staticDir string) http.HandlerFunc {
	fs := http.FileServer(http.Dir(staticDir))

	return func(w http.ResponseWriter, r *http.Request) {
		// Don't serve the index.html for API requests
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// Check if the requested file exists
		path := filepath.Join(staticDir, r.URL.Path)

		fileInfo, err := os.Stat(path)
		if err != nil {
			log.Printf("File error: %s, %v", path, err)
		}

		// If the file doesn't exist or is a directory, serve index.html
		if os.IsNotExist(err) || (r.URL.Path != "/" && (strings.HasSuffix(r.URL.Path, "/") || (err == nil && fileInfo.IsDir()))) {
			indexPath := filepath.Join(staticDir, "index.html")

			http.ServeFile(w, r, indexPath)
			return
		}

		fs.ServeHTTP(w, r)
	}
}
