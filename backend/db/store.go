package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"sloggo/models"
	"sloggo/utils"

	_ "github.com/mattn/go-sqlite3"
)

var (
	dbDirectory    string
	dbInstance     *sql.DB
	batchMutex     sync.Mutex
	batchLogs      []string
	batchParams    [][]any
	maxBatchSize   = 10000
	checkpointTick = 5 * time.Second
	cleanupTick    = 15 * time.Minute
)

// Using the shared LogEntry from models package

// ChartDataPoint represents a single point of log data for charts
type ChartDataPoint struct {
	Timestamp int64 `json:"timestamp"`
	Debug     int   `json:"debug"`
	Info      int   `json:"info"`
	Notice    int   `json:"notice"`
	Warning   int   `json:"warning"`
	Error     int   `json:"error"`
	Critical  int   `json:"critical"`
	Alert     int   `json:"alert"`
	Emergency int   `json:"emergency"`
}

// FacetMetadata represents metadata for faceted search
type FacetMetadata struct {
	Rows  []FacetRow `json:"rows"`
	Total int        `json:"total"`
	Min   *int       `json:"min,omitempty"`
	Max   *int       `json:"max,omitempty"`
}

// FacetRow represents a single row in facet metadata
type FacetRow struct {
	Value any `json:"value"`
	Total int `json:"total"`
}

func init() {
	// Set up database connection
	setupDatabase()

	// Initialize schema
	query := `
	CREATE TABLE IF NOT EXISTS logs (
	    facility INTEGER NOT NULL,
	    severity INTEGER NOT NULL,
	    version INTEGER NOT NULL DEFAULT 1,
	    timestamp TEXT NOT NULL,
	    hostname TEXT NOT NULL,
	    app_name TEXT NOT NULL,
	    procid TEXT,
	    msgid TEXT,
	    structured_data TEXT,
	    msg TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_logs_hostname ON logs(hostname);
	CREATE INDEX IF NOT EXISTS idx_logs_app_name ON logs(app_name);
	CREATE INDEX IF NOT EXISTS idx_logs_facility ON logs(facility);
	CREATE INDEX IF NOT EXISTS idx_logs_severity ON logs(severity);
	`

	if _, err := dbInstance.Exec(query); err != nil {
		log.Fatalf("Failed to create logs table: %v", err)
	}

	log.Println("Logs table created or already exists")

	// Initialize the batch logs slice
	batchLogs = make([]string, 0, maxBatchSize)
	batchParams = make([][]any, 0, maxBatchSize)

	// Start the batch processor
	go processBatchPeriodically()

	// Start the checkpoint process
	go performCheckpointsPeriodically()

	// Start the log cleanup process
	go performLogCleanupPeriodically()
}

// GetDBInstance returns the initialized SQLite database instance.
func GetDBInstance() *sql.DB {
	return dbInstance
}

// StoreLog adds a log message to the batch for efficient processing
func StoreLog(query string, params []any) error {
	batchMutex.Lock()
	defer batchMutex.Unlock()

	batchLogs = append(batchLogs, query)
	batchParams = append(batchParams, params)

	// If we've reached the max batch size, process immediately
	if len(batchLogs) >= maxBatchSize {
		return processBatch()
	}

	return nil
}

// ForceProcessBatch forces immediate processing of the batch queue
// This is primarily used for testing to ensure logs are written to the database
func ForceProcessBatch() error {
	batchMutex.Lock()
	defer batchMutex.Unlock()
	return processBatch()
}

// processBatch processes all pending log entries in a single transaction
func processBatch() error {
	if len(batchLogs) == 0 {
		return nil
	}

	// Start a transaction
	tx, err := dbInstance.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return err
	}

	// Prepare to execute each statement
	for i := 0; i < len(batchLogs); i++ {
		_, err := tx.Exec(batchLogs[i], batchParams[i]...)
		if err != nil {
			tx.Rollback()
			log.Printf("Failed to execute batch statement: %v", err)
			return err
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		return err
	}

	// Clear the batches
	batchLogs = batchLogs[:0]
	batchParams = batchParams[:0]

	return nil
}

// processBatchPeriodically processes any pending logs on a timer
func processBatchPeriodically() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		batchMutex.Lock()
		err := processBatch()
		batchMutex.Unlock()

		if err != nil {
			log.Printf("Error in periodic batch processing: %v", err)
		}
	}
}

// performCheckpoint executes a checkpoint to flush WAL to the main database file
func performCheckpoint() error {
	_, err := dbInstance.Exec("PRAGMA wal_checkpoint(PASSIVE);")
	if err != nil {
		log.Printf("Failed to perform WAL checkpoint: %v", err)
		return err
	}
	return nil
}

// performCheckpointsPeriodically runs checkpoints on a timer
func performCheckpointsPeriodically() {
	ticker := time.NewTicker(checkpointTick)
	defer ticker.Stop()

	for range ticker.C {
		if err := performCheckpoint(); err != nil {
			log.Printf("Error in periodic checkpoint: %v", err)
		}
	}
}

// cleanupOldLogs deletes logs older than the retention period
func cleanupOldLogs() error {
	// Calculate the cutoff timestamp for deletion (current time - retention period)
	cutoffTime := time.Now().Add(-time.Duration(utils.LogRetentionMinutes) * time.Minute).UTC().Format(time.RFC3339Nano)

	// Create a query to delete logs older than the cutoff timestamp
	query := "DELETE FROM logs WHERE timestamp < ?"

	// Execute the deletion
	result, err := dbInstance.Exec(query, cutoffTime)
	if err != nil {
		log.Printf("Failed to delete old logs: %v", err)
		return err
	}

	// Log the number of deleted rows
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Failed to get rows affected by cleanup: %v", err)
	} else if rowsAffected > 0 {
		log.Printf("Cleaned up %d log entries older than %s", rowsAffected, cutoffTime)
	}

	return nil
}

// performLogCleanupPeriodically runs log cleanup on a timer
func performLogCleanupPeriodically() {
	ticker := time.NewTicker(cleanupTick)
	defer ticker.Stop()

	// Run cleanup immediately at startup
	if err := cleanupOldLogs(); err != nil {
		log.Printf("Error in initial log cleanup: %v", err)
	}

	for range ticker.C {
		if err := cleanupOldLogs(); err != nil {
			log.Printf("Error in periodic log cleanup: %v", err)
		}
	}
}

// GetLogs retrieves logs from the database based on filters
func GetLogs(limit int, cursor time.Time, direction string, filters map[string]any, sortField string, sortOrder string) ([]models.LogEntry, int, int, error) {
	// Build query
	queryBuilder := strings.Builder{}
	countQueryBuilder := strings.Builder{}
	filterQueryBuilder := strings.Builder{}
	args := []any{}

	// Start the main query
	queryBuilder.WriteString("SELECT rowid, facility, severity, timestamp, hostname, app_name, procid, msgid, structured_data, msg FROM logs ")

	// Start the count query
	countQueryBuilder.WriteString("SELECT COUNT(*) FROM logs ")

	// Process query with pagination parameters

	// Build WHERE clause for filtering
	whereClause := buildWhereClause(filters, cursor, direction, &args)
	if whereClause != "" {
		filterQueryBuilder.WriteString("WHERE ")
		filterQueryBuilder.WriteString(whereClause)
	}

	// Apply the filter clause to both queries
	queryBuilder.WriteString(filterQueryBuilder.String())
	countQueryBuilder.WriteString(filterQueryBuilder.String())

	// Add sorting
	if sortField != "" && sortOrder != "" {
		queryBuilder.WriteString(fmt.Sprintf(" ORDER BY %s %s", sortField, sortOrder))
	} else {
		queryBuilder.WriteString(" ORDER BY timestamp DESC")
	}

	// Add limit
	queryBuilder.WriteString(fmt.Sprintf(" LIMIT %d", limit))

	// Execute the count query to get total filtered rows
	var filterCount int
	err := dbInstance.QueryRow(countQueryBuilder.String(), args...).Scan(&filterCount)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("error counting filtered logs: %v", err)
	}

	// Get total row count
	var totalCount int
	err = dbInstance.QueryRow("SELECT COUNT(*) FROM logs").Scan(&totalCount)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("error counting total logs: %v", err)
	}

	// Execute the main query
	rows, err := dbInstance.Query(queryBuilder.String(), args...)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("error querying logs: %v", err)
	}
	defer rows.Close()

	// Parse results
	logs := []models.LogEntry{}
	for rows.Next() {
		var entry models.LogEntry
		var timestampStr string

		err := rows.Scan(
			&entry.RowID,
			&entry.Facility,
			&entry.Severity,
			&timestampStr,
			&entry.Hostname,
			&entry.AppName,
			&entry.ProcID,
			&entry.MsgID,
			&entry.StructuredData,
			&entry.Message,
		)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("error scanning log row: %v", err)
		}

		// Parse timestamp
		entry.Timestamp, err = time.Parse(time.RFC3339Nano, timestampStr)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("error parsing timestamp: %v", err)
		}

		logs = append(logs, entry)
	}

	// Process logs retrieved from database

	// If direction is "prev", we need to reverse the order of the logs
	if direction == "prev" {
		// Reverse the order
		for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
			logs[i], logs[j] = logs[j], logs[i]
		}
	}

	// If no logs were found and we're using specific time constraints, try a fallback
	if len(logs) == 0 && !cursor.IsZero() {
		// No logs found with cursor, falling back to most recent logs

		// Clear the args and build a new query to get the most recent logs
		args = []any{}
		queryBuilder.Reset()
		queryBuilder.WriteString("SELECT rowid, facility, severity, timestamp, hostname, app_name, procid, msgid, structured_data, msg FROM logs ")

		// Use only the non-timestamp filters
		modifiedFilters := make(map[string]any)
		for k, v := range filters {
			if k != "startDate" && k != "endDate" {
				modifiedFilters[k] = v
			}
		}

		// Build a where clause without cursor constraints
		whereClause = buildWhereClause(modifiedFilters, time.Time{}, "", &args)
		if whereClause != "" {
			queryBuilder.WriteString("WHERE ")
			queryBuilder.WriteString(whereClause)
		}

		// Order by timestamp DESC to get most recent logs
		queryBuilder.WriteString(" ORDER BY timestamp DESC LIMIT ?")
		args = append(args, limit)

		// Execute the fallback query
		fallbackRows, err := dbInstance.Query(queryBuilder.String(), args...)
		if err != nil {
			// Continue with empty results if fallback fails
		} else {
			defer fallbackRows.Close()

			// Clear previous logs and parse fallback results
			logs = []models.LogEntry{}
			for fallbackRows.Next() {
				var entry models.LogEntry
				var timestampStr string

				err := fallbackRows.Scan(
					&entry.RowID,
					&entry.Facility,
					&entry.Severity,
					&timestampStr,
					&entry.Hostname,
					&entry.AppName,
					&entry.ProcID,
					&entry.MsgID,
					&entry.StructuredData,
					&entry.Message,
				)
				if err != nil {
					// Skip entries with scanning errors
					continue
				}

				// Parse timestamp
				entry.Timestamp, err = time.Parse(time.RFC3339Nano, timestampStr)
				if err != nil {
					// Skip entries with timestamp parsing errors
					continue
				}

				logs = append(logs, entry)
			}

			// Fallback query completed
		}
	}

	return logs, totalCount, filterCount, nil
}

// GetFacets retrieves facet metadata for filtering
func GetFacets(filters map[string]any) (map[string]FacetMetadata, error) {
	facets := make(map[string]FacetMetadata)

	// Get hostname facets
	hostnameRows, err := getFacetValues("hostname", filters, 20)
	if err != nil {
		return nil, err
	}
	facets["hostname"] = FacetMetadata{
		Rows:  hostnameRows,
		Total: len(hostnameRows),
	}

	// Get app_name facets
	appNameRows, err := getFacetValues("app_name", filters, 20)
	if err != nil {
		return nil, err
	}
	facets["appName"] = FacetMetadata{
		Rows:  appNameRows,
		Total: len(appNameRows),
	}

	// Get facility facets
	facilityRows, err := getFacetValues("facility", filters, 24)
	if err != nil {
		return nil, err
	}
	facets["facility"] = FacetMetadata{
		Rows:  facilityRows,
		Total: len(facilityRows),
	}

	// Get severity facets
	severityRows, err := getFacetValues("severity", filters, 8)
	if err != nil {
		return nil, err
	}
	facets["severity"] = FacetMetadata{
		Rows:  severityRows,
		Total: len(severityRows),
	}

	// Get procid facets
	procidRows, err := getFacetValues("procid", filters, 20)
	if err != nil {
		return nil, err
	}
	facets["procId"] = FacetMetadata{
		Rows:  procidRows,
		Total: len(procidRows),
	}

	// Get msgid facets
	msgidRows, err := getFacetValues("msgid", filters, 20)
	if err != nil {
		return nil, err
	}
	facets["msgId"] = FacetMetadata{
		Rows:  msgidRows,
		Total: len(msgidRows),
	}

	// Get priority min/max
	minMaxPriority, err := getMinMaxValues("facility * 8 + severity", filters)
	if err != nil {
		return nil, err
	}
	facets["priority"] = FacetMetadata{
		Rows:  []FacetRow{},
		Total: 0,
		Min:   minMaxPriority[0],
		Max:   minMaxPriority[1],
	}

	return facets, nil
}

// GetChartData retrieves time-series data for charts
func GetChartData(filters map[string]any) ([]ChartDataPoint, error) {
	// Define time ranges for chart data (e.g., last 24 hours with hourly points)
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	// Build query for chart data
	queryBuilder := strings.Builder{}
	args := []any{}

	queryBuilder.WriteString(`
		SELECT
			strftime('%s', timestamp) * 1000 as ts,
			SUM(CASE WHEN severity = 7 THEN 1 ELSE 0 END) as debug,
			SUM(CASE WHEN severity = 6 THEN 1 ELSE 0 END) as info,
			SUM(CASE WHEN severity = 5 THEN 1 ELSE 0 END) as notice,
			SUM(CASE WHEN severity = 4 THEN 1 ELSE 0 END) as warning,
			SUM(CASE WHEN severity = 3 THEN 1 ELSE 0 END) as error,
			SUM(CASE WHEN severity = 2 THEN 1 ELSE 0 END) as critical,
			SUM(CASE WHEN severity = 1 THEN 1 ELSE 0 END) as alert,
			SUM(CASE WHEN severity = 0 THEN 1 ELSE 0 END) as emergency
		FROM logs
	`)

	// Add WHERE clause for filtering
	whereClause := buildWhereClause(filters, time.Time{}, "", &args)
	if whereClause != "" {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(whereClause)

		// Add time range for chart
		queryBuilder.WriteString(" AND timestamp BETWEEN ? AND ?")
		args = append(args, startTime.Format(time.RFC3339Nano), endTime.Format(time.RFC3339Nano))
	} else {
		queryBuilder.WriteString(" WHERE timestamp BETWEEN ? AND ?")
		args = append(args, startTime.Format(time.RFC3339Nano), endTime.Format(time.RFC3339Nano))
	}

	// Group by hour
	queryBuilder.WriteString(" GROUP BY strftime('%Y-%m-%d %H', timestamp) ORDER BY ts ASC")

	// Execute query
	rows, err := dbInstance.Query(queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("error querying chart data: %v", err)
	}
	defer rows.Close()

	// Parse results
	chartData := []ChartDataPoint{}
	for rows.Next() {
		var point ChartDataPoint
		err := rows.Scan(
			&point.Timestamp,
			&point.Debug,
			&point.Info,
			&point.Notice,
			&point.Warning,
			&point.Error,
			&point.Critical,
			&point.Alert,
			&point.Emergency,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning chart data row: %v", err)
		}

		chartData = append(chartData, point)
	}

	return chartData, nil
}

// Helper function to get facet values
func getFacetValues(field string, filters map[string]any, limit int) ([]FacetRow, error) {
	queryBuilder := strings.Builder{}
	args := []any{}

	queryBuilder.WriteString(fmt.Sprintf("SELECT %s as value, COUNT(*) as total FROM logs", field))

	// Add WHERE clause for filtering, excluding filter on the current field
	tempFilters := make(map[string]any)
	for k, v := range filters {
		if k != field && k != fmt.Sprintf("%sMin", field) && k != fmt.Sprintf("%sMax", field) {
			tempFilters[k] = v
		}
	}

	whereClause := buildWhereClause(tempFilters, time.Time{}, "", &args)
	if whereClause != "" {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(whereClause)
	}

	queryBuilder.WriteString(fmt.Sprintf(" GROUP BY %s ORDER BY total DESC LIMIT %d", field, limit))

	// Execute query
	rows, err := dbInstance.Query(queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("error querying facet values for %s: %v", field, err)
	}
	defer rows.Close()

	// Parse results
	facetRows := []FacetRow{}
	for rows.Next() {
		var row FacetRow
		var valueStr string
		err := rows.Scan(&valueStr, &row.Total)
		if err != nil {
			return nil, fmt.Errorf("error scanning facet row: %v", err)
		}

		// Try to convert to integer if possible
		if intVal, err := strconv.Atoi(valueStr); err == nil {
			row.Value = intVal
		} else {
			row.Value = valueStr
		}

		facetRows = append(facetRows, row)
	}

	return facetRows, nil
}

// Helper function to get min/max values for a field
func getMinMaxValues(field string, filters map[string]any) ([]*int, error) {
	queryBuilder := strings.Builder{}
	args := []any{}

	// Fix the SQL syntax by removing the "as" keyword in the field parameter
	// and using direct column aliases instead
	queryBuilder.WriteString(fmt.Sprintf("SELECT MIN(%s) as min_val, MAX(%s) as max_val FROM logs", field, field))

	whereClause := buildWhereClause(filters, time.Time{}, "", &args)
	if whereClause != "" {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(whereClause)
	}

	// Execute query
	var minVal, maxVal sql.NullInt64
	err := dbInstance.QueryRow(queryBuilder.String(), args...).Scan(&minVal, &maxVal)
	if err != nil {
		return nil, fmt.Errorf("error querying min/max values for %s: %v", field, err)
	}

	result := []*int{nil, nil}
	if minVal.Valid {
		min := int(minVal.Int64)
		result[0] = &min
	}
	if maxVal.Valid {
		max := int(maxVal.Int64)
		result[1] = &max
	}

	return result, nil
}

// Helper function to build WHERE clause from filters
func buildWhereClause(filters map[string]any, cursor time.Time, direction string, args *[]any) string {
	conditions := []string{}

	// Add timestamp condition for pagination
	if !cursor.IsZero() {
		// For "prev" direction, we want logs before the cursor time
		// For "next" direction, we want logs after the cursor time
		if direction == "prev" {
			conditions = append(conditions, "timestamp <= ?")
		} else {
			conditions = append(conditions, "timestamp >= ?")
		}
		*args = append(*args, cursor.Format(time.RFC3339Nano))
	}

	// Add filter conditions
	for key, value := range filters {
		switch key {
		case "level":
			levels := value.([]string)
			if len(levels) > 0 {
				levelConditions := []string{}
				for _, level := range levels {
					var severity int
					switch level {
					case "emergency":
						severity = 0
					case "alert":
						severity = 1
					case "critical":
						severity = 2
					case "error":
						severity = 3
					case "warning":
						severity = 4
					case "notice":
						severity = 5
					case "info":
						severity = 6
					case "debug":
						severity = 7
					}
					levelConditions = append(levelConditions, "severity = ?")
					*args = append(*args, severity)
				}
				conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(levelConditions, " OR ")))
			}
		case "hostname":
			conditions = append(conditions, "hostname LIKE ?")
			*args = append(*args, fmt.Sprintf("%%%s%%", value.(string)))
		case "appName":
			conditions = append(conditions, "app_name LIKE ?")
			*args = append(*args, fmt.Sprintf("%%%s%%", value.(string)))
		case "procId":
			conditions = append(conditions, "procid LIKE ?")
			*args = append(*args, fmt.Sprintf("%%%s%%", value.(string)))
		case "msgId":
			conditions = append(conditions, "msgid LIKE ?")
			*args = append(*args, fmt.Sprintf("%%%s%%", value.(string)))
		case "facility":
			facilities := value.([]int)
			if len(facilities) > 0 {
				placeholders := make([]string, len(facilities))
				for i, f := range facilities {
					placeholders[i] = "?"
					*args = append(*args, f)
				}
				conditions = append(conditions, fmt.Sprintf("facility IN (%s)", strings.Join(placeholders, ",")))
			}
		case "severity":
			severities := value.([]int)
			if len(severities) > 0 {
				placeholders := make([]string, len(severities))
				for i, s := range severities {
					placeholders[i] = "?"
					*args = append(*args, s)
				}
				conditions = append(conditions, fmt.Sprintf("severity IN (%s)", strings.Join(placeholders, ",")))
			}
		case "priorityMin":
			conditions = append(conditions, "facility * 8 + severity >= ?")
			*args = append(*args, value.(int))
		case "priorityMax":
			conditions = append(conditions, "facility * 8 + severity <= ?")
			*args = append(*args, value.(int))
		case "startDate":
			conditions = append(conditions, "timestamp >= ?")
			*args = append(*args, value.(time.Time).Format(time.RFC3339Nano))
		case "endDate":
			conditions = append(conditions, "timestamp <= ?")
			*args = append(*args, value.(time.Time).Format(time.RFC3339Nano))
		}
	}

	return strings.Join(conditions, " AND ")
}

// setupDatabase initializes the database connection
// Uses in-memory database for tests and file-based for production
func setupDatabase() {
	var err error
	var dbPath string

	if testing.Testing() {
		dbPath = ":memory:"
	} else {
		e, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}

		dbPath = filepath.Join(path.Dir(e), ".sqlite/logs.db")
	}

	dbInstance, err = sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_cache_size=10000&_busy_timeout=5000")
	if err != nil {
		log.Fatalf("Failed to connect to SQLite database: %v", err)
	}

	// Set connection pool parameters
	dbInstance.SetMaxOpenConns(1)
	dbInstance.SetMaxIdleConns(1)
	dbInstance.SetConnMaxLifetime(10 * time.Minute)
}
