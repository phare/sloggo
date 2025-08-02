package server

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// StartHTTPServer initializes and starts the HTTP server, serving both the API and the React frontend.
func StartHTTPServer() {
	frontendDir := "/app/public"
	fs := http.FileServer(http.Dir(frontendDir))

	// Serve React frontend and ensure frontend assets are served
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(frontendDir, r.URL.Path)
		if _, err := os.Stat(path); os.IsNotExist(err) || filepath.Ext(r.URL.Path) == "" {
			http.ServeFile(w, r, filepath.Join(frontendDir, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	}))

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Sloggo backend is running"))
	})

	log.Println("HTTP server is running on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Failed to start HTTP server:", err)
	}
}
