package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
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

	"github.com/marcboeker/go-duckdb/v2"
)

var (
	db                    *sql.DB
	batchLogsMutex        sync.Mutex
	batchLogs             []models.LogEntry
	maxBatchStoreLogsSize = 10000
	cleanupTick           = 30 * time.Minute
)

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
	Rows []FacetRow `json:"rows"`
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
	setupDatabaseTable("logs")

	batchLogs = make([]models.LogEntry, 0, maxBatchStoreLogsSize)

	// Start the batch processor
	go processBatchPeriodically()

	// Start the log cleanup process
	go performLogCleanupPeriodically()
}

// setupDatabase initializes the database connections
// Uses in-memory database for tests and file-based for production
func setupDatabase() {
	var err error

	e, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}

	dsn := filepath.Join(path.Dir(e), ".duckdb/logs.db")

	if testing.Testing() {
		dsn = ""
	}

	db, err = sql.Open("duckdb", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
}

// setupDatabaseTable creates a table if it doesn't already exist
func setupDatabaseTable(table string) {
	query := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
	    severity INTEGER NOT NULL,
	    facility INTEGER NOT NULL,
	    version INTEGER NOT NULL DEFAULT 1,
	    timestamp TIMESTAMP NOT NULL,
	    hostname TEXT NOT NULL,
	    app_name TEXT NOT NULL,
	    procid TEXT,
	    msgid TEXT,
	    structured_data TEXT,
	    msg TEXT
	);
	`, table)

	if _, err := db.Exec(query); err != nil {
		log.Fatalf("Failed to create table %s: %v", table, err)
	}
}

// GetDBInstance returns the initialized DuckDB database instance.
func GetDBInstance() *sql.DB {
	return db
}

// StoreLog adds a log entry to the batch for efficient processing
func StoreLog(entry models.LogEntry) error {
	batchLogsMutex.Lock()
	batchLogs = append(batchLogs, entry)

	// If we've reached the max batch size, process immediately
	if len(batchLogs) >= maxBatchStoreLogsSize {
		batchLogsMutex.Unlock()
		return ProcessBatchStoreLogs()
	}

	batchLogsMutex.Unlock()
	return nil
}

// processBatchStoreLogsUnsafe processes all pending log entries without acquiring mutex
// Must be called with batchStoreLogsMutex already held
func ProcessBatchStoreLogs() error {
	batchLogsMutex.Lock()
	if len(batchLogs) == 0 {
		batchLogsMutex.Unlock()
		return nil
	}

	entries := batchLogs
	batchLogs = batchLogs[:0]

	batchLogsMutex.Unlock()

	// Get the underlying DuckDB connection from sql.DB
	dbConn, err := db.Conn(context.Background())
	if err != nil {
		return err
	}
	defer dbConn.Close()

	var rawConn driver.Conn
	err = dbConn.Raw(func(driverConn any) error {
		rawConn = driverConn.(driver.Conn)
		return nil
	})
	if err != nil {
		return err
	}

	appender, err := duckdb.NewAppenderFromConn(rawConn, "", "logs")
	if err != nil {
		log.Printf("Failed to create appender: %v", err)
		return err
	}
	defer func() {
		if closeErr := appender.Close(); closeErr != nil {
			log.Printf("Error closing appender: %v", closeErr)
		}
	}()

	// Append each log entry directly from struct fields
	for i, entry := range entries {
		if err := appender.AppendRow(
			entry.Severity,
			entry.Facility,
			entry.Version,
			entry.Timestamp,
			entry.Hostname,
			entry.AppName,
			entry.ProcID,
			entry.MsgID,
			entry.StructuredData,
			entry.Message,
		); err != nil {
			log.Printf("Failed to append row %d: %v", i+1, err)
			return err
		}
	}

	// Flush the appender to ensure data is written
	if err := appender.Flush(); err != nil {
		log.Printf("Failed to flush appender: %v", err)
		return err
	}
	return nil
}

// processBatchPeriodically processes any pending logs on a timer
func processBatchPeriodically() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := ProcessBatchStoreLogs(); err != nil {
			log.Printf("Error in periodic batch processing: %v", err)
		}
	}
}

// cleanupOldLogs deletes logs older than the retention period
func cleanupOldLogs() error {
	// Calculate the cutoff timestamp for deletion (current time - retention period)
	cutoffTime := time.Now().Add(-time.Duration(utils.LogRetentionMinutes) * time.Minute).UTC().Format(time.RFC3339Nano)

	query := "DELETE FROM logs WHERE timestamp < ?"

	result, err := db.Exec(query, cutoffTime)
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

	for range ticker.C {
		if err := cleanupOldLogs(); err != nil {
			log.Printf("Error in periodic log cleanup: %v", err)
		}
	}
}

// GetLogs retrieves logs from the database based on filters
func GetLogs(limit int, cursor time.Time, direction string, filters map[string]any, sortField string, sortOrder string) ([]models.LogEntry, error) {
	// Build query
	queryBuilder := strings.Builder{}
	filterQueryBuilder := strings.Builder{}
	args := []any{}

	queryBuilder.WriteString("SELECT rowid, facility, severity, timestamp, hostname, app_name, procid, msgid, structured_data, msg FROM logs ")

	whereClause := buildWhereClause(filters, cursor, direction, &args)
	if whereClause != "" {
		filterQueryBuilder.WriteString("WHERE ")
		filterQueryBuilder.WriteString(whereClause)
	}

	queryBuilder.WriteString(filterQueryBuilder.String())

	if sortField != "" && sortOrder != "" {
		queryBuilder.WriteString(fmt.Sprintf(" ORDER BY %s %s", sortField, sortOrder))
	} else {
		queryBuilder.WriteString(" ORDER BY timestamp DESC")
	}

	queryBuilder.WriteString(fmt.Sprintf(" LIMIT %d", limit))

	rows, err := readDbInstance.Query(queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("error querying logs: %v", err)
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
			return nil, fmt.Errorf("error scanning log row: %v", err)
		}

		// Parse timestamp
		entry.Timestamp, err = time.Parse(time.RFC3339Nano, timestampStr)
		if err != nil {
			return nil, fmt.Errorf("error parsing timestamp: %v", err)
		}

		logs = append(logs, entry)
	}

	return logs, nil
}

// GetFacets retrieves facet metadata for filtering
func GetFacets(filters map[string]any) (map[string]FacetMetadata, error) {
	// For facets, exclude temporal filters (date range) to show total state
	// This ensures live mode facets represent all logs, not just new ones
	facetFilters := make(map[string]any)
	for k, v := range filters {
		if k != "startDate" && k != "endDate" {
			facetFilters[k] = v
		}
	}

	facets := make(map[string]FacetMetadata)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var globalErr error

	// Fast direct queries in parallel
	wg.Add(2)

	// Get severity facets concurrently with highly optimized query
	go func() {
		defer wg.Done()

		query := "SELECT severity as value, COUNT(*) as total FROM logs"
		args := []any{}

		whereClause := buildWhereClause(facetFilters, time.Time{}, "", &args)
		if whereClause != "" {
			query += " WHERE " + whereClause
		}

		query += " GROUP BY severity"

		rows, err := db.Query(query, args...)
		if err != nil {
			mu.Lock()
			globalErr = fmt.Errorf("error querying severity facets: %v", err)
			mu.Unlock()
			return
		}
		defer rows.Close()

		facetRows := []FacetRow{}
		for rows.Next() {
			var row FacetRow
			var valueStr string
			err := rows.Scan(&valueStr, &row.Total)
			if err != nil {
				mu.Lock()
				globalErr = fmt.Errorf("error scanning severity facet row: %v", err)
				mu.Unlock()
				return
			}

			// Try to convert to integer if possible
			if intVal, err := strconv.Atoi(valueStr); err == nil {
				row.Value = intVal
			} else {
				row.Value = valueStr
			}

			facetRows = append(facetRows, row)
		}

		mu.Lock()
		facets["severity"] = FacetMetadata{
			Rows: facetRows,
		}
		mu.Unlock()
	}()

	// Get facility facets concurrently
	go func() {
		defer wg.Done()

		query := "SELECT facility as value, COUNT(*) as total FROM logs"
		args := []any{}

		whereClause := buildWhereClause(facetFilters, time.Time{}, "", &args)
		if whereClause != "" {
			query += " WHERE " + whereClause
		}

		query += " GROUP BY facility"

		rows, err := db.Query(query, args...)
		if err != nil {
			mu.Lock()
			globalErr = fmt.Errorf("error querying facility facets: %v", err)
			mu.Unlock()
			return
		}
		defer rows.Close()

		facetRows := []FacetRow{}
		for rows.Next() {
			var row FacetRow
			var valueStr string
			err := rows.Scan(&valueStr, &row.Total)
			if err != nil {
				mu.Lock()
				globalErr = fmt.Errorf("error scanning facility facet row: %v", err)
				mu.Unlock()
				return
			}

			// Try to convert to integer if possible
			if intVal, err := strconv.Atoi(valueStr); err == nil {
				row.Value = intVal
			} else {
				row.Value = valueStr
			}

			facetRows = append(facetRows, row)
		}

		mu.Lock()
		facets["facility"] = FacetMetadata{
			Rows: facetRows,
		}
		mu.Unlock()
	}()

	// Wait for all goroutines to complete
	wg.Wait()

	// Check if any errors occurred
	if globalErr != nil {
		return nil, globalErr
	}

	return facets, nil
}

// GetChartData retrieves time-series data for charts
func GetChartData(cursor time.Time, filters map[string]any) ([]ChartDataPoint, error) {
	chartFilters := make(map[string]any)
	for k, v := range filters {
		chartFilters[k] = v
	}

	// If date filters are not provided, we use the cursor set to the next hour as the end
	// time for chart data and go back 24 hours to get the last 24 hours of data.
	if chartFilters["startDate"] == nil || chartFilters["endDate"] == nil {
		endDate := cursor.Truncate(time.Hour).Add(time.Hour)
		startDate := cursor.Add(-24 * time.Hour)
		chartFilters["endDate"] = endDate
		chartFilters["startDate"] = startDate
	}

	startDate := chartFilters["startDate"].(time.Time)
	endDate := chartFilters["endDate"].(time.Time)
	duration := endDate.Sub(startDate)

	var truncateUnit string

	switch {
	case duration <= 3*24*time.Hour: // Up to 3 days: group by hour (max 72 points)
		truncateUnit = "hour"
	case duration <= 21*24*time.Hour: // Up to 3 weeks: group by day (max 21 points)
		truncateUnit = "day"
	case duration <= 180*24*time.Hour: // Up to ~6 months: group by week (max 26 points)
		truncateUnit = "week"
	default: // More than 6 months: group by month
		truncateUnit = "month"
	}

	// Build query for chart data
	queryBuilder := strings.Builder{}
	args := []any{}

	queryBuilder.WriteString(fmt.Sprintf(`
		SELECT
		    CAST(epoch(date_trunc('%s', timestamp)) * 1000 AS BIGINT) AS ts,
			SUM(CASE WHEN severity = 7 THEN 1 ELSE 0 END) as debug,
			SUM(CASE WHEN severity = 6 THEN 1 ELSE 0 END) as info,
			SUM(CASE WHEN severity = 5 THEN 1 ELSE 0 END) as notice,
			SUM(CASE WHEN severity = 4 THEN 1 ELSE 0 END) as warning,
			SUM(CASE WHEN severity = 3 THEN 1 ELSE 0 END) as error,
			SUM(CASE WHEN severity = 2 THEN 1 ELSE 0 END) as critical,
			SUM(CASE WHEN severity = 1 THEN 1 ELSE 0 END) as alert,
			SUM(CASE WHEN severity = 0 THEN 1 ELSE 0 END) as emergency
		FROM logs
	`, truncateUnit))

	// Add WHERE clause for filtering (excluding temporal constraints)
	whereClause := buildWhereClause(chartFilters, time.Time{}, "", &args)
	if whereClause != "" {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(whereClause)
	}

	// Group by hour
	queryBuilder.WriteString(fmt.Sprintf(" GROUP BY date_trunc('%s', timestamp) ORDER BY ts ASC", truncateUnit))

	// Execute query
	rows, err := db.Query(queryBuilder.String(), args...)
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

// Helper function to build WHERE clause from filters
func buildWhereClause(filters map[string]any, cursor time.Time, direction string, args *[]any) string {
	if len(filters) == 0 && cursor.IsZero() {
		return ""
	}

	conditions := make([]string, 0, len(filters)+1)

	// Add filter conditions
	for key, value := range filters {
		switch key {
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
		case "hostname":
			conditions = append(conditions, "hostname = ?")
			*args = append(*args, value.(string))
		case "procId":
			conditions = append(conditions, "procid = ?")
			*args = append(*args, value.(string))
		case "appName":
			conditions = append(conditions, "app_name = ?")
			*args = append(*args, value.(string))
		case "msgId":
			conditions = append(conditions, "msgid = ?")
			*args = append(*args, value.(string))
		case "startDate":
			conditions = append(conditions, "timestamp >= ?")
			*args = append(*args, value.(time.Time).Format(time.RFC3339Nano))
		case "endDate":
			conditions = append(conditions, "timestamp <= ?")
			*args = append(*args, value.(time.Time).Format(time.RFC3339Nano))
		}
	}

	if !cursor.IsZero() {
		if direction == "prev" {
			conditions = append(conditions, "timestamp > ?")
		} else {
			conditions = append(conditions, "timestamp < ?")
		}
		*args = append(*args, cursor.Format(time.RFC3339Nano))
	}

	return strings.Join(conditions, " AND ")
}
