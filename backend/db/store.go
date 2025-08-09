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

	"github.com/mattn/go-sqlite3"
	"github.com/qustavo/sqlhooks/v2"
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

	sqlDriver := "sqlite3"

	if utils.Debug {
		sqlDriver = "sqlite3hooks"
		sql.Register(sqlDriver, sqlhooks.Wrap(&sqlite3.SQLiteDriver{}, &Hooks{}))
	}

	if testing.Testing() {
		dbPath = ":memory:"
	} else {
		e, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}

		dbPath = filepath.Join(path.Dir(e), ".sqlite/logs.db")
	}

	connectionString := dbPath + "?_journal_mode=WAL&_synchronous=NORMAL&_cache_size=10000&_busy_timeout=5000"

	// Write connection pool - single connection for write operations
	writeDbInstance, err = sql.Open(sqlDriver, connectionString)
	if err != nil {
		log.Fatalf("Failed to connect to SQLite write database: %v", err)
	}

	// Set write connection pool parameters - single connection
	writeDbInstance.SetMaxOpenConns(1)
	writeDbInstance.SetMaxIdleConns(1)
	writeDbInstance.SetConnMaxLifetime(30 * time.Minute)

	// Read connection pool - multiple connections for read operations
	readDbInstance, err = sql.Open(sqlDriver, connectionString)
	if err != nil {
		log.Fatalf("Failed to connect to SQLite read database: %v", err)
	}

	// Set read connection pool parameters - multiple connections
	readDbInstance.SetMaxOpenConns(10)
	readDbInstance.SetMaxIdleConns(5)
	readDbInstance.SetConnMaxLifetime(30 * time.Minute)
}

// setupDatabaseTable creates a table if it doesn't already exist
func setupDatabaseTable(table string) {
	query := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
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

	-- Add indexes for common filter combinations
	CREATE INDEX IF NOT EXISTS idx_timestamp ON %s(timestamp);
	CREATE INDEX IF NOT EXISTS idx_hostname ON %s(hostname);
	CREATE INDEX IF NOT EXISTS idx_app_name ON %s(app_name);
	CREATE INDEX IF NOT EXISTS idx_facility ON %s(facility);
	CREATE INDEX IF NOT EXISTS idx_severity ON %s(severity);
	CREATE INDEX IF NOT EXISTS idx_severity_timestamp ON %s(severity, timestamp);
	CREATE INDEX IF NOT EXISTS idx_facility_timestamp ON %s(facility, timestamp);
	CREATE INDEX IF NOT EXISTS idx_severity_facility_timestamp ON %s(severity, facility, timestamp);
	CREATE INDEX IF NOT EXISTS idx_hostname_timestamp ON %s(hostname, timestamp);
	CREATE INDEX IF NOT EXISTS idx_app_name_timestamp ON %s(app_name, timestamp);
	`, table, table, table, table, table, table, table, table, table, table, table)

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

	// Execute combined count query to get both filtered and total counts in one query
	var filterCount, totalCount int
	combinedCountQuery := fmt.Sprintf("SELECT (%s) as filtered_count, (SELECT COUNT(*) FROM logs) as total_count", countQueryBuilder.String())
	err := readDbInstance.QueryRow(combinedCountQuery, args...).Scan(&filterCount, &totalCount)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("error counting logs: %v", err)
	}

	// Execute the main query
	rows, err := readDbInstance.Query(queryBuilder.String(), args...)
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

	return logs, totalCount, filterCount, nil
}

// GetFacets retrieves facet metadata for filtering
func GetFacets(filters map[string]any) (map[string]FacetMetadata, error) {
	facets := make(map[string]FacetMetadata)

	// For facets, exclude temporal filters (date range) to show total state
	// This ensures live mode facets represent all logs, not just new ones
	facetFilters := make(map[string]any)
	for k, v := range filters {
		if k != "startDate" && k != "endDate" {
			facetFilters[k] = v
		}
	}

	// Calculate total count for facets based on non-temporal filters only
	var totalCount int
	countQuery := "SELECT COUNT(*) FROM logs"
	countArgs := []any{}
	whereClause := buildWhereClause(facetFilters, time.Time{}, "", &countArgs)
	if whereClause != "" {
		countQuery += " WHERE " + whereClause
	}

	err := dbInstance.QueryRow(countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("error counting total filtered rows for facets: %v", err)
	}

	// Get severity facets
	severityRows, err := getFacetValues("severity", facetFilters, 8)
	if err != nil {
		return nil, err
	}
	facets["severity"] = FacetMetadata{
		Rows:  severityRows,
		Total: totalCount,
	}

	// Get facility facets
	facilityRows, err := getFacetValues("facility", facetFilters, 24)
	if err != nil {
		return nil, err
	}
	facets["facility"] = FacetMetadata{
		Rows:  facilityRows,
		Total: totalCount,
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
	chartFilters["endDate"] = cursor.Truncate(time.Hour).Add(time.Hour)
	chartFilters["startDate"] = cursor.Add(-24 * time.Hour)

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

// Helper function to get facet values
func getFacetValues(field string, filters map[string]any, limit int) ([]FacetRow, error) {
	queryBuilder := strings.Builder{}
	args := []any{}

	queryBuilder.WriteString(fmt.Sprintf("SELECT %s as value, COUNT(*) as total FROM logs", field))

	// Add WHERE clause for filtering, including all filters
	whereClause := buildWhereClause(filters, time.Time{}, "", &args)
	if whereClause != "" {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(whereClause)
	}

	queryBuilder.WriteString(fmt.Sprintf(" GROUP BY %s ORDER BY total DESC LIMIT %d", field, limit))

	// Execute query
	rows, err := readDbInstance.Query(queryBuilder.String(), args...)
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
			conditions = append(conditions, "hostname LIKE ?")
			*args = append(*args, fmt.Sprintf("%%%s%%", value.(string)))
		case "appName":
			conditions = append(conditions, "app_name LIKE ?")
			*args = append(*args, fmt.Sprintf("%%%s%%", value.(string)))
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
