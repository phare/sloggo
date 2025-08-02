package server

import (
	"log"
	"net/http"
	"os"
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
