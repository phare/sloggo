package server

import (
	"log"
	"net/http"
	"os"
	"sloggo/server/handlers"
	"sloggo/utils"
)

type Server struct {
	port   string
	server *http.Server
}

func (s *Server) setupRoutes() {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/api/health", handlers.HealthHandler)

	// API endpoint for logs
	mux.HandleFunc("/api/logs", handlers.LogsHandler)

	// Serve static files from the frontend build
	staticDir := "/app/public"
	mux.Handle("/", handlers.StaticHandler(staticDir))

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

// NewServer creates a new HTTP server instance
func NewServer() *Server {
	// Use environment variable for port if available
	port := os.Getenv("SLOGGO_API_PORT")
	if port == "" {
		port = utils.ApiPort
	}

	return &Server{
		port: port,
	}
}

// StartHTTPServer initializes and starts the HTTP server
func StartHTTPServer() {
	server := NewServer()

	if err := server.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatal("Failed to start HTTP server:", err)
	}
}
