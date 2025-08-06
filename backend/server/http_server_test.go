package server

import (
	"encoding/json"
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
		name           string
		path           string
		method         string
		expectedCode   int
		expectedBody   string
		checkJSONValid bool
	}{
		{
			name:         "Health check returns 200",
			path:         "/api/health",
			method:       "GET",
			expectedCode: http.StatusOK,
			expectedBody: "Sloggo backend is running",
		},
		{
			name:           "Logs endpoint returns valid JSON",
			path:           "/api/logs",
			method:         "GET",
			expectedCode:   http.StatusOK,
			checkJSONValid: true,
		},
		{
			name:           "Logs endpoint with filter parameters",
			path:           "/api/logs?severity=emergency,warning&hostname=testhost",
			method:         "GET",
			expectedCode:   http.StatusOK,
			checkJSONValid: true,
		},
		{
			name:           "Logs endpoint with pagination parameters",
			path:           "/api/logs?size=10&cursor=1628097603000",
			method:         "GET",
			expectedCode:   http.StatusOK,
			checkJSONValid: true,
		},
		{
			name:           "Logs endpoint with sort parameters",
			path:           "/api/logs?sort=timestamp.asc",
			method:         "GET",
			expectedCode:   http.StatusOK,
			checkJSONValid: true,
		},
		{
			name:         "Logs endpoint with method not allowed",
			path:         "/api/logs",
			method:       "POST",
			expectedCode: http.StatusMethodNotAllowed,
		},
		{
			name:         "Non-existent API endpoint returns 404",
			path:         "/api/nonexistent",
			method:       "GET",
			expectedCode: http.StatusNotFound,
		},
	}

	server := NewServer()
	server.setupRoutes()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()

			server.server.Handler.ServeHTTP(w, req)

			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != tc.expectedCode {
				t.Errorf("Expected status code %d, got %d", tc.expectedCode, resp.StatusCode)
			}

			if tc.expectedBody != "" && string(body) != tc.expectedBody {
				t.Errorf("Expected body %q, got %q", tc.expectedBody, string(body))
			}

			if tc.checkJSONValid {
				// Check if the response is valid JSON
				var result map[string]interface{}
				err := json.Unmarshal(body, &result)
				if err != nil {
					t.Errorf("Invalid JSON response: %v", err)
				}

				// Check for required fields in the logs response
				if _, ok := result["data"]; !ok {
					t.Error("JSON response missing 'data' field")
				}
				if _, ok := result["meta"]; !ok {
					t.Error("JSON response missing 'meta' field")
				}
			}
		})
	}
}

func TestServerIntegration(t *testing.T) {
	// Set custom port for testing
	testPort := "8081"
	os.Setenv("SLOGGO_API_PORT", testPort)
	defer os.Unsetenv("SLOGGO_API_PORT")

	// Create a test server
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
	testIntegrationEndpoints := []struct {
		name           string
		path           string
		method         string
		expectedCode   int
		expectedBody   string
		checkJSONValid bool
	}{
		{
			name:         "Health check returns 200",
			path:         "/api/health",
			method:       "GET",
			expectedCode: http.StatusOK,
			expectedBody: "Sloggo backend is running",
		},
		{
			name:           "Logs endpoint returns valid JSON",
			path:           "/api/logs",
			method:         "GET",
			expectedCode:   http.StatusOK,
			checkJSONValid: true,
		},
		{
			name:           "Logs endpoint with filter parameters",
			path:           "/api/logs?severity=emergency,warning",
			method:         "GET",
			expectedCode:   http.StatusOK,
			checkJSONValid: true,
		},
		{
			name:           "Logs endpoint with pagination parameters",
			path:           "/api/logs?size=5",
			method:         "GET",
			expectedCode:   http.StatusOK,
			checkJSONValid: true,
		},
	}

	for _, tc := range testIntegrationEndpoints {
		t.Run(tc.name, func(t *testing.T) {
			var resp *http.Response
			var err error

			url := fmt.Sprintf("http://localhost:%s%s", testPort, tc.path)

			if tc.method == "GET" {
				resp, err = http.Get(url)
			} else if tc.method == "POST" {
				resp, err = http.Post(url, "application/json", nil)
			} else {
				req, _ := http.NewRequest(tc.method, url, nil)
				client := &http.Client{}
				resp, err = client.Do(req)
			}

			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedCode {
				t.Errorf("Expected status %d, got %d", tc.expectedCode, resp.StatusCode)
			}

			body, _ := io.ReadAll(resp.Body)

			if tc.expectedBody != "" && string(body) != tc.expectedBody {
				t.Errorf("Expected body %q, got %q", tc.expectedBody, string(body))
			}

			if tc.checkJSONValid {
				var result map[string]interface{}
				if err := json.Unmarshal(body, &result); err != nil {
					t.Errorf("Endpoint returned invalid JSON: %v", err)
					return
				}

				// Check for required fields in the logs response
				if _, ok := result["data"]; !ok {
					t.Error("JSON response missing 'data' field")
				}
				if _, ok := result["meta"]; !ok {
					t.Error("JSON response missing 'meta' field")
				}

				// Check meta structure
				if meta, ok := result["meta"].(map[string]interface{}); ok {
					if _, ok := meta["totalRowCount"]; !ok {
						t.Error("Meta is missing 'totalRowCount' field")
					}
					if _, ok := meta["filterRowCount"]; !ok {
						t.Error("Meta is missing 'filterRowCount' field")
					}
					if _, ok := meta["chartData"]; !ok {
						t.Error("Meta is missing 'chartData' field")
					}
					if _, ok := meta["facets"]; !ok {
						t.Error("Meta is missing 'facets' field")
					}
				}
			}
		})
	}

	// Test method not allowed
	postReq, _ := http.NewRequest("POST", fmt.Sprintf("http://localhost:%s/api/logs", testPort), nil)
	client := &http.Client{}
	postResp, err := client.Do(postReq)
	if err != nil {
		t.Fatalf("Failed to make POST request: %v", err)
	}
	defer postResp.Body.Close()

	if postResp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for POST request, got %d", postResp.StatusCode)
	}

	// Test non-existent endpoint
	notFoundResp, err := http.Get(fmt.Sprintf("http://localhost:%s/api/nonexistent", testPort))
	if err != nil {
		t.Fatalf("Failed to make request to non-existent endpoint: %v", err)
	}
	defer notFoundResp.Body.Close()

	if notFoundResp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent endpoint, got %d", notFoundResp.StatusCode)
	}

	// Cleanup
	server.Shutdown()
}

// Test creating a mock server and test the handler directly
func TestMockServer(t *testing.T) {
	server := NewServer()
	server.setupRoutes()

	ts := httptest.NewServer(server.server.Handler)
	defer ts.Close()

	// Test CORS headers
	req, _ := http.NewRequest("OPTIONS", ts.URL+"/api/logs", nil)
	req.Header.Set("Origin", "http://example.com")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make OPTIONS request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for OPTIONS request, got %d", resp.StatusCode)
	}

	// Check CORS headers
	corsHeaders := []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
	}

	for _, header := range corsHeaders {
		if resp.Header.Get(header) == "" {
			t.Errorf("CORS header %s not set", header)
		}
	}
}
