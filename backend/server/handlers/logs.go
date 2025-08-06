package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sloggo/db"
	"sloggo/models"
	"strconv"
	"strings"
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

	// Get logs from database
	logs, totalCount, filterCount, err := db.GetLogs(size, cursor, direction, filters, sortField, sortOrder)
	if err != nil {
		log.Printf("Error fetching logs: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
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
		logs[i].ParsedStructuredData = structData

		// Ensure timestamp is properly formatted for JavaScript to parse
		// This is already handled by Go's JSON marshaller, but making it explicit
		if logs[i].Timestamp.IsZero() {
			logs[i].Timestamp = time.Now()
		}
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

	// Set content type and encode response
	w.Header().Set("Content-Type", "application/json")

	// Send the response to the client
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
