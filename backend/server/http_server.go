package server

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Server struct {
	port   string
	server *http.Server
}

func NewServer() *Server {
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}
	return &Server{
		port: port,
	}
}

func (s *Server) setupRoutes() {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Sloggo backend is running"))
	})

	// API endpoints would go here
	// mux.HandleFunc("/api/...", ...)

	// Serve static files from the frontend build
	staticDir := "/app/public"
	fs := http.FileServer(http.Dir(staticDir))

	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

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
	}))

	s.server = &http.Server{
		Addr:    ":" + s.port,
		Handler: mux,
	}
}

func (s *Server) Start() error {
	s.setupRoutes()

	log.Printf("HTTP server is running on :%s", s.port)
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// StartHTTPServer initializes and starts the HTTP server
func StartHTTPServer() {
	server := NewServer()
	if err := server.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatal("Failed to start HTTP server:", err)
	}
}
