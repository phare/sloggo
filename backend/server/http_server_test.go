package server

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Health check returns 200",
			path:         "/health",
			expectedCode: http.StatusOK,
			expectedBody: "Sloggo backend is running",
		},
	}

	server := NewServer()
	server.setupRoutes()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			w := httptest.NewRecorder()

			server.server.Handler.ServeHTTP(w, req)

			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != tc.expectedCode {
				t.Errorf("Expected status code %d, got %d", tc.expectedCode, resp.StatusCode)
			}

			if string(body) != tc.expectedBody {
				t.Errorf("Expected body %q, got %q", tc.expectedBody, string(body))
			}
		})
	}
}

func TestServerIntegration(t *testing.T) {
	// Set custom port for testing
	testPort := "8081"
	os.Setenv("HTTP_PORT", testPort)
	defer os.Unsetenv("HTTP_PORT")

	server := NewServer()
	go func() {
		err := server.Start()
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Test real HTTP request
	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/health", testPort))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "Sloggo backend is running" {
		t.Errorf("Expected 'Sloggo backend is running', got %q", string(body))
	}

	// Cleanup
	server.Shutdown()
}
