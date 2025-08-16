package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sloggo/db"
	"sloggo/models"
	"sloggo/utils"
	"strconv"
	"strings"
	"sync"
	"time"
)

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
	Metadata       map[string]any              `json:"metadata,omitempty"`
}

// LogsHandler handles the API endpoint for logs
func LogsHandler(w http.ResponseWriter, r *http.Request) {
	requestStartTime := time.Now()

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
	size := 50

	if sizeStr := query.Get("size"); sizeStr != "" {
		if parsedSize, err := strconv.Atoi(sizeStr); err == nil && parsedSize > 0 {
			size = parsedSize
		}
	}

	// Direction for pagination
	direction := query.Get("direction")
	if direction == "" {
		direction = "next"
	} else if direction != "next" && direction != "prev" {
		direction = "next"
	}

	// Filters
	filters := make(map[string]any)

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

	// Parse cursor (timestamp) for pagination
	var cursor time.Time
	now := time.Now().UTC().Add(1 * time.Minute) // Allow for clock skew

	if cursorStr := query.Get("cursor"); cursorStr != "" {
		if parsedCursor, err := strconv.ParseInt(cursorStr, 10, 64); err == nil {
			cursorTime := time.Unix(0, parsedCursor*int64(time.Millisecond))
			if cursorTime.After(now) {
				cursor = now
			} else {
				cursor = cursorTime
			}
		} else {
			// Use current time if parsing fails
			cursor = now
		}
	} else {
		// Default to current time if no cursor provided
		cursor = now
	}

	// Date range filter
	if dateStr := query.Get("timestamp"); dateStr != "" {
		dateValues := strings.Split(dateStr, "-")

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

	// Parallelize database calls for better performance
	var wg sync.WaitGroup
	var logs []models.LogEntry
	var totalCount, filterCount int
	var facets map[string]db.FacetMetadata
	var chartData []db.ChartDataPoint
	var logsErr, facetsErr, chartErr error

	wg.Add(3)

	// Time for all database operations
	queryStartTime := time.Now()

	// Get logs from database
	go func() {
		defer wg.Done()
		logs, totalCount, filterCount, logsErr = db.GetLogs(size, cursor, direction, filters, sortField, sortOrder)

		if utils.Debug {
			log.Printf("⚡ GetLogs execution time: %v", time.Since(queryStartTime))
		}
	}()

	// Get facets for filtering
	go func() {
		defer wg.Done()
		facets, facetsErr = db.GetFacets(filters)

		if utils.Debug {
			log.Printf("⚡ GetFacets execution time: %v", time.Since(queryStartTime))
		}
	}()

	// Get chart data
	go func() {
		defer wg.Done()
		chartData, chartErr = db.GetChartData(cursor, filters)

		if utils.Debug {
			log.Printf("⚡️ GetChartData execution time: %v", time.Since(queryStartTime))
		}
	}()

	// Wait for all goroutines to complete
	wg.Wait()
	if utils.Debug {
		log.Printf("⚡️ Total database operations execution time: %v", time.Since(queryStartTime))
	}
	// Check for errors
	if logsErr != nil {
		log.Printf("Error fetching logs: %v", logsErr)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if facetsErr != nil {
		log.Printf("Error fetching facets: %v", facetsErr)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if chartErr != nil {
		log.Printf("Error fetching chart data: %v", chartErr)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Process logs for API response format
	processStartTime := time.Now()
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
		logs[i].ParsedStructuredData = structData

		// Ensure timestamp is properly formatted for JavaScript to parse
		// This is already handled by Go's JSON marshaller, but making it explicit
		if logs[i].Timestamp.IsZero() {
			logs[i].Timestamp = time.Now()
		}
	}

	if utils.Debug {
		log.Printf("⚡️ Log processing time: %v", time.Since(processStartTime))
	}
	// Determine next and previous cursors
	var nextCursor, prevCursor *int64 = nil, nil
	if len(logs) > 0 {
		nextVal := logs[len(logs)-1].Timestamp.UnixNano() / int64(time.Millisecond)
		prevVal := logs[0].Timestamp.UnixNano() / int64(time.Millisecond)
		nextCursor = &nextVal
		prevCursor = &prevVal
	}

	// Prepare the response
	prepareResponseStartTime := time.Now()
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

	if utils.Debug {
		log.Printf("⚡️ Response preparation time: %v", time.Since(prepareResponseStartTime))
	}

	// Set content type and encode response
	w.Header().Set("Content-Type", "application/json")

	// Send the response to the client
	encodeStartTime := time.Now()
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if utils.Debug {
		log.Printf("⚡️ JSON encoding time: %v", time.Since(encodeStartTime))
		log.Printf("⚡️ Total request handling time: %v\n\n", time.Since(requestStartTime))
	}
}
