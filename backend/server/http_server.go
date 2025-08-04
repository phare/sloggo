package server

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sloggo/db"
	"sloggo/models"
	"sloggo/utils"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	port   string
	server *http.Server
}

// LogsResponse represents the API response format for logs
type LogsResponse struct {
	Data       []models.LogEntry `json:"data"`
	Meta       InfiniteQueryMeta `json:"meta"`
	NextCursor *int64            `json:"nextCursor"`
	PrevCursor *int64            `json:"prevCursor"`
}

// InfiniteQueryMeta contains metadata for infinite scrolling
type InfiniteQueryMeta struct {
	TotalRowCount  int                         `json:"totalRowCount"`
	FilterRowCount int                         `json:"filterRowCount"`
	ChartData      []db.ChartDataPoint         `json:"chartData"`
	Facets         map[string]db.FacetMetadata `json:"facets"`
	Metadata       map[string]interface{}      `json:"metadata,omitempty"`
}

// Using db.ChartDataPoint type instead of redefining it here
// Using db.FacetMetadata type instead of redefining it here
// Using db.FacetRow type instead of redefining it here

// Using models.GetSeverityLevel() instead of a local function

// No longer needed - removed debug logging function

func (s *Server) setupRoutes() {
	mux := http.NewServeMux()

	// Health check endpoint - moved to /api/health
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Sloggo backend is running"))
	})

	// API endpoint for logs
	mux.HandleFunc("/api/logs", func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for cross-origin requests in development
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse query parameters
		query := r.URL.Query()

		// Pagination parameters
		size := 40
		if sizeStr := query.Get("size"); sizeStr != "" {
			if parsedSize, err := strconv.Atoi(sizeStr); err == nil && parsedSize > 0 {
				size = parsedSize
			}
		}

		// Parse cursor (timestamp) for pagination
		var cursor time.Time
		if cursorStr := query.Get("cursor"); cursorStr != "" {
			if parsedCursor, err := strconv.ParseInt(cursorStr, 10, 64); err == nil {
				cursor = time.Unix(0, parsedCursor*int64(time.Millisecond))
			} else {
				// Use current time if parsing fails
				cursor = time.Now()
			}
		} else {
			// Default to current time if no cursor provided
			cursor = time.Now()
		}

		// Direction for pagination
		direction := query.Get("direction")
		if direction == "" {
			direction = "next"
		}

		// Filters
		filters := make(map[string]interface{})

		// Level filter
		if levelStr := query.Get("level"); levelStr != "" {
			filters["level"] = strings.Split(levelStr, ",")
		}

		// Hostname filter
		if hostname := query.Get("hostname"); hostname != "" {
			filters["hostname"] = hostname
		}

		// App name filter
		if appName := query.Get("appName"); appName != "" {
			filters["appName"] = appName
		}

		// Process ID filter
		if procId := query.Get("procId"); procId != "" {
			filters["procId"] = procId
		}

		// Message ID filter
		if msgId := query.Get("msgId"); msgId != "" {
			filters["msgId"] = msgId
		}

		// Facility filter
		if facilityStr := query.Get("facility"); facilityStr != "" {
			facilityValues := strings.Split(facilityStr, ",")
			facilities := make([]int, 0, len(facilityValues))
			for _, v := range facilityValues {
				if facility, err := strconv.Atoi(v); err == nil {
					facilities = append(facilities, facility)
				}
			}
			if len(facilities) > 0 {
				filters["facility"] = facilities
			}
		}

		// Severity filter
		if severityStr := query.Get("severity"); severityStr != "" {
			severityValues := strings.Split(severityStr, ",")
			severities := make([]int, 0, len(severityValues))
			for _, v := range severityValues {
				if severity, err := strconv.Atoi(v); err == nil {
					severities = append(severities, severity)
				}
			}
			if len(severities) > 0 {
				filters["severity"] = severities
			}
		}

		// Priority filter
		if priorityStr := query.Get("priority"); priorityStr != "" {
			priorityValues := strings.Split(priorityStr, "-")
			if len(priorityValues) == 2 {
				minPriority, minErr := strconv.Atoi(priorityValues[0])
				maxPriority, maxErr := strconv.Atoi(priorityValues[1])
				if minErr == nil && maxErr == nil {
					filters["priorityMin"] = minPriority
					filters["priorityMax"] = maxPriority
				}
			}
		}

		// Date range filter
		if dateStr := query.Get("date"); dateStr != "" {
			dateValues := strings.Split(dateStr, "..")
			if len(dateValues) == 2 {
				startMillis, startErr := strconv.ParseInt(dateValues[0], 10, 64)
				endMillis, endErr := strconv.ParseInt(dateValues[1], 10, 64)
				if startErr == nil && endErr == nil {
					filters["startDate"] = time.Unix(0, startMillis*int64(time.Millisecond))
					filters["endDate"] = time.Unix(0, endMillis*int64(time.Millisecond))
				}
			}
		}

		// Sort parameter
		sortField := "timestamp"
		sortOrder := "DESC"
		if sortStr := query.Get("sort"); sortStr != "" {
			sortParts := strings.Split(sortStr, ".")
			if len(sortParts) == 2 {
				sortField = sortParts[0]
				if sortParts[1] == "asc" {
					sortOrder = "ASC"
				}
			}
		}

		// Get logs from database
		logs, totalCount, filterCount, err := db.GetLogs(size, cursor, direction, filters, sortField, sortOrder)
		if err != nil {
			log.Printf("Error fetching logs: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// If no logs were returned and we're using 'prev' direction (which is for initial load),
		// try again with 'next' direction from the oldest possible time to get at least some data
		// If no logs were found and we're using 'prev' direction (which is for initial load),
		// try again with 'next' direction from the oldest possible time to get at least some data
		if len(logs) == 0 && direction == "prev" {
			oldCursor := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC) // Unix epoch start
			logs, totalCount, filterCount, err = db.GetLogs(size, oldCursor, "next", filters, sortField, sortOrder)
			if err != nil {
				log.Printf("Error fetching logs with fallback: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
		}

		// Get facets for filtering
		facets, err := db.GetFacets(filters)
		if err != nil {
			log.Printf("Error fetching facets: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Get chart data
		chartData, err := db.GetChartData(filters)
		if err != nil {
			log.Printf("Error fetching chart data: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Process logs for API response format
		for i := range logs {
			// Parse structured data JSON if present
			structData := make(map[string]map[string]string)

			if logs[i].StructuredData != "" && logs[i].StructuredData != "-" {
				// Attempt to parse the JSON data
				if err := json.Unmarshal([]byte(logs[i].StructuredData), &structData); err != nil {
					log.Printf("Error parsing structured data for row %d", logs[i].RowID)
				}
			}

			// Calculate priority
			logs[i].Priority = logs[i].Facility*8 + logs[i].Severity
			logs[i].Level = models.GetSeverityLevel(logs[i].Severity)
			logs[i].ParsedStructuredData = structData

			// Ensure timestamp is properly formatted for JavaScript to parse
			// This is already handled by Go's JSON marshaller, but making it explicit
			if logs[i].Timestamp.IsZero() {
				logs[i].Timestamp = time.Now()
			}
		}

		// Process logs completed

		// Determine next and previous cursors
		var nextCursor, prevCursor *int64 = nil, nil
		if len(logs) > 0 {
			if direction == "next" {
				nextVal := logs[len(logs)-1].Timestamp.UnixNano() / int64(time.Millisecond)
				prevVal := logs[0].Timestamp.UnixNano() / int64(time.Millisecond)
				nextCursor = &nextVal
				prevCursor = &prevVal
			} else {
				nextVal := logs[0].Timestamp.UnixNano() / int64(time.Millisecond)
				prevVal := logs[len(logs)-1].Timestamp.UnixNano() / int64(time.Millisecond)
				nextCursor = &nextVal
				prevCursor = &prevVal
			}

			// Cursors are now set properly
		}

		// Prepare the response
		response := LogsResponse{
			Data: logs,
			Meta: InfiniteQueryMeta{
				TotalRowCount:  totalCount,
				FilterRowCount: filterCount,
				ChartData:      chartData,
				Facets:         facets,
				Metadata:       map[string]any{},
			},
			NextCursor: nextCursor,
			PrevCursor: prevCursor,
		}

		// Response is ready to send

		// Set content type and encode response
		w.Header().Set("Content-Type", "application/json")

		// Send the response to the client

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	})

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
