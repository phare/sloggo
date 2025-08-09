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
	dbDirectory           string
	writeDbInstance       *sql.DB
	readDbInstance        *sql.DB
	batchStoreLogsMutex   sync.Mutex
	batchStoreLogsParams  [][]any
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

// facetCacheEntry represents a cached facet result with expiration
type facetCacheEntry struct {
	data      map[string]FacetMetadata
	timestamp time.Time
}

func init() {
	// Set up database connection
	setupDatabase()

	// Initialize schema
	setupDatabaseTable("logs")

	batchStoreLogsParams = make([][]any, 0, maxBatchStoreLogsSize)

	// Start the batch processor
	go processBatchPeriodically()

	// Start the log cleanup process
	go performLogCleanupPeriodically()
}

// setupDatabase initializes the database connections
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

	base := "file:" + dbPath

	writeDSN := base + "?_journal_mode=WAL" +
		"&_synchronous=NORMAL" +
		"&_busy_timeout=5000" +
		"&_cache_size=-8192" + // ~8 MiB cache for the writer
		"&_wal_autocheckpoint=4000" + // auto-checkpoint ~16 MiB WAL if page size is 4 KiB
		"&_journal_size_limit=134217728" + // cap WAL journal to 128 MiB
		"&_temp_store=MEMORY" // keep small temps in RAM

	readDSN := base + "?mode=ro" + // ensure read-only
		"&_query_only=1" + // prevent accidental writes
		"&_busy_timeout=5000" +
		"&_cache_size=-4096" + // ~4 MiB per read connection
		"&_mmap_size=134217728" // 128 MiB memory map for faster reads (virtual memory; not pre-allocated)

	// Write connection pool - single connection for write operations
	writeDbInstance, err = sql.Open("sqlite3", writeDSN)
	if err != nil {
		log.Fatalf("Failed to connect to SQLite write database: %v", err)
	}

	// Set write connection pool parameters - single connection
	writeDbInstance.SetMaxOpenConns(1)
	writeDbInstance.SetMaxIdleConns(1)
	writeDbInstance.SetConnMaxLifetime(0)

	// Read connection pool - multiple connections for read operations
	readDbInstance, err = sql.Open("sqlite3", readDSN)
	if err != nil {
		log.Fatalf("Failed to connect to SQLite read database: %v", err)
	}

	// Set read connection pool parameters - multiple connections
	readDbInstance.SetMaxOpenConns(5)
	readDbInstance.SetMaxIdleConns(2)
	readDbInstance.SetConnMaxLifetime(0)
}

// setupDatabaseTable creates a table if it doesn't already exist
func setupDatabaseTable(table string) {
	query := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
	    severity INTEGER NOT NULL,
	    facility INTEGER NOT NULL,
	    version INTEGER NOT NULL DEFAULT 1,
	    timestamp TEXT NOT NULL,
	    hostname TEXT NOT NULL,
	    app_name TEXT NOT NULL,
	    procid TEXT,
	    msgid TEXT,
	    structured_data TEXT,
	    msg TEXT
	);

	-- Add indexes for common filter combinations
	CREATE INDEX IF NOT EXISTS idx_timestamp ON %s(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_logs_dynamic ON %s(severity, facility, hostname, procid, app_name, msgid);
	CREATE INDEX IF NOT EXISTS idx_severity_timestamp ON %s(severity, timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_facility_timestamp ON %s(facility, timestamp DESC);

	-- Run ANALYZE to collect statistics for query planning
	ANALYZE;
	`, table, table, table, table, table)

	if _, err := writeDbInstance.Exec(query); err != nil {
		log.Fatalf("Failed to create table %s: %v", table, err)
	}
}

// GetDBInstance returns the initialized SQLite database instance.
func GetDBInstance() *sql.DB {
	return writeDbInstance
}

// StoreLog adds a log message to the batch for efficient processing
func StoreLog(params []any) error {
	batchStoreLogsMutex.Lock()
	defer batchStoreLogsMutex.Unlock()

	batchStoreLogsParams = append(batchStoreLogsParams, params)

	// If we've reached the max batch size, process immediately
	if len(batchStoreLogsParams) >= maxBatchStoreLogsSize {
		return processBatchStoreLogs()
	}

	return nil
}

// ForceProcessBatchStoreLogs forces immediate processing of the batch queue
// This is primarily used for testing to ensure logs are written to the database
func ForceProcessBatchStoreLogs() error {
	batchStoreLogsMutex.Lock()
	defer batchStoreLogsMutex.Unlock()
	return processBatchStoreLogs()
}

// processBatchStoreLogs processes all pending log entries in a single transaction
func processBatchStoreLogs() error {
	var insertStatement *sql.Stmt

	if len(batchStoreLogsParams) == 0 {
		return nil
	}

	insertStatement, err := writeDbInstance.Prepare(`
		INSERT INTO logs (
			facility, severity, version, timestamp,
			hostname, app_name, procid, msgid,
			structured_data, msg
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)

	if err != nil {
		log.Printf("Failed to prepare INSERT statement: %v", err)
		return err
	}
	defer insertStatement.Close()

	transaction, err := writeDbInstance.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return err
	}

	transactionStatement := transaction.Stmt(insertStatement)
	defer transactionStatement.Close()

	// Execute each parameter set
	for _, params := range batchStoreLogsParams {
		_, err := transactionStatement.Exec(params...)
		if err != nil {
			transaction.Rollback()
			log.Printf("Failed to execute batch statement: %v", err)
			return err
		}
	}

	if err := transaction.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		return err
	}

	batchStoreLogsParams = nil
	return nil
}

// processBatchPeriodically processes any pending logs on a timer
func processBatchPeriodically() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		batchStoreLogsMutex.Lock()
		err := processBatchStoreLogs()
		batchStoreLogsMutex.Unlock()

		if err != nil {
			log.Printf("Error in periodic batch processing: %v", err)
		}
	}
}

// cleanupOldLogs deletes logs older than the retention period
func cleanupOldLogs() error {
	// Calculate the cutoff timestamp for deletion (current time - retention period)
	cutoffTime := time.Now().Add(-time.Duration(utils.LogRetentionMinutes) * time.Minute).UTC().Format(time.RFC3339Nano)

	query := "DELETE FROM logs WHERE timestamp < ?"

	result, err := writeDbInstance.Exec(query, cutoffTime)
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

		// Force SQLite to use the severity index by putting it first in the select list
		query := "SELECT severity as value, COUNT(*) as total FROM logs"
		args := []any{}

		whereClause := buildWhereClause(facetFilters, time.Time{}, "", &args)
		if whereClause != "" {
			query += " WHERE " + whereClause
		}

		query += " GROUP BY severity"

		rows, err := readDbInstance.Query(query, args...)
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

		// Force SQLite to use the facility index by putting it first in the select list
		query := "SELECT facility as value, COUNT(*) as total FROM logs"
		args := []any{}

		whereClause := buildWhereClause(facetFilters, time.Time{}, "", &args)
		if whereClause != "" {
			query += " WHERE " + whereClause
		}

		query += " GROUP BY facility"

		rows, err := readDbInstance.Query(query, args...)
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

	// We always use the cursor set to the next hour as the end time for chart data
	// and go back 24 hours to get the last 24 hours of data.
	endDate := cursor.Truncate(time.Hour).Add(time.Hour)
	startDate := cursor.Add(-24 * time.Hour)
	chartFilters["endDate"] = endDate
	chartFilters["startDate"] = startDate

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

	// Add WHERE clause for filtering (excluding temporal constraints)
	whereClause := buildWhereClause(chartFilters, time.Time{}, "", &args)
	if whereClause != "" {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(whereClause)
	}

	// Group by hour
	queryBuilder.WriteString(" GROUP BY strftime('%Y-%m-%d %H', timestamp) ORDER BY ts ASC")

	// Execute query
	rows, err := readDbInstance.Query(queryBuilder.String(), args...)
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

	if !cursor.IsZero() {
		if direction == "prev" {
			conditions = append(conditions, "timestamp > ?")
		} else {
			conditions = append(conditions, "timestamp < ?")
		}
		*args = append(*args, cursor.Format(time.RFC3339Nano))
	}

	// Add filter conditions
	for key, value := range filters {
		switch key {
		case "hostname":
			conditions = append(conditions, "hostname = ?")
			*args = append(*args, value.(string))
		case "appName":
			conditions = append(conditions, "app_name = ?")
			*args = append(*args, value.(string))
		case "procId":
			conditions = append(conditions, "procid = ?")
			*args = append(*args, value.(string))
		case "msgId":
			conditions = append(conditions, "msgid = ?")
			*args = append(*args, value.(string))
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
