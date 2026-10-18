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
		// Don't unlock here - let ProcessBatchStoreLogs handle it
		// by calling it while holding the lock
		entries := batchLogs
		batchLogs = make([]models.LogEntry, 0, maxBatchStoreLogsSize)
		batchLogsMutex.Unlock()

		// Process the batch outside the lock
		return processBatchStoreLogsWithEntries(entries)
	}

	batchLogsMutex.Unlock()
	return nil
}

// ProcessBatchStoreLogs processes all pending log entries
// This is called by the periodic batch processor
func ProcessBatchStoreLogs() error {
	batchLogsMutex.Lock()
	if len(batchLogs) == 0 {
		batchLogsMutex.Unlock()
		return nil
	}

	entries := batchLogs
	batchLogs = make([]models.LogEntry, 0, maxBatchStoreLogsSize)
	batchLogsMutex.Unlock()

	return processBatchStoreLogsWithEntries(entries)
}

// processBatchStoreLogsWithEntries processes a batch of log entries
// This function does not touch the global batchLogs slice
func processBatchStoreLogsWithEntries(entries []models.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

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

// logSortColumns maps the logical sort field used by the API to the physical
// column name in the logs table.
var logSortColumns = map[string]string{
	"timestamp":      "timestamp",
	"severity":       "severity",
	"facility":       "facility",
	"hostname":       "hostname",
	"appName":        "app_name",
	"procId":         "procid",
	"msgId":          "msgid",
	"message":        "msg",
	"structuredData": "structured_data",
}

// Cursor is a keyset pagination position: the value of the sort column for the
// boundary row of a page plus its rowid. The rowid breaks ties between rows
// sharing the same sort value (e.g. identical timestamps), which is required
// for stable pagination.
type Cursor struct {
	Value string
	RowID int64
}

// ParseCursor decodes a cursor returned by the API. It returns false when the
// input is not a valid cursor.
func ParseCursor(s string) (Cursor, bool) {
	i := strings.LastIndex(s, ":")
	if i < 0 {
		return Cursor{}, false
	}
	rowID, err := strconv.ParseInt(s[i+1:], 10, 64)
	if err != nil {
		return Cursor{}, false
	}
	return Cursor{Value: s[:i], RowID: rowID}, true
}

func formatCursor(value string, rowID int64) string {
	return value + ":" + strconv.FormatInt(rowID, 10)
}

// cursorValue returns the sort column value of an entry in the string form used
// by the cursor.
func cursorValue(entry models.LogEntry, sortField string) string {
	switch sortField {
	case "severity":
		return strconv.Itoa(int(entry.Severity))
	case "facility":
		return strconv.Itoa(int(entry.Facility))
	case "hostname":
		return entry.Hostname
	case "appName":
		return entry.AppName
	case "procId":
		return entry.ProcID
	case "msgId":
		return entry.MsgID
	case "message":
		return entry.Message
	case "structuredData":
		return entry.StructuredData
	default:
		return strconv.FormatInt(entry.Timestamp.UnixNano()/int64(time.Millisecond), 10)
	}
}

// cursorArg converts a cursor value into the argument expected by the sort
// column, so that the keyset comparison uses matching types.
func cursorArg(sortField, value string) any {
	switch sortField {
	case "severity", "facility":
		if n, err := strconv.Atoi(value); err == nil {
			return n
		}
	case "timestamp":
		if ms, err := strconv.ParseInt(value, 10, 64); err == nil {
			return time.Unix(0, ms*int64(time.Millisecond)).UTC().Format(time.RFC3339Nano)
		}
	}
	return value
}

// GetLogs retrieves a page of logs using keyset (cursor) pagination.
//
// The sort key is (sort column, rowid): rowid is a stable tiebreaker that keeps
// pagination consistent when several rows share the same sort value, which is
// common for syslog (identical timestamps). The returned cursors are opaque
// strings that must be passed back verbatim.
func GetLogs(limit int, cursor *Cursor, direction string, filters map[string]any, sortField string, sortOrder string) ([]models.LogEntry, int, int, *string, *string, error) {
	column, ok := logSortColumns[sortField]
	if !ok {
		sortField, column = "timestamp", "timestamp"
	}
	if sortOrder != "ASC" {
		sortOrder = "DESC"
	}

	// Row counts are independent of the pagination cursor: filterRowCount is the
	// number of rows matching the filters, totalCount the whole table.
	var countArgs []any
	countClause := buildWhereClause(filters, nil, sortField, sortOrder, direction, &countArgs)
	countQuery := "SELECT COUNT(*) FROM logs"
	if countClause != "" {
		countQuery += " WHERE " + countClause
	}
	var filterCount, totalCount int
	combinedCountQuery := fmt.Sprintf("SELECT (%s) as filtered_count, (SELECT COUNT(*) FROM logs) as total_count", countQuery)
	if err := db.QueryRow(combinedCountQuery, countArgs...).Scan(&filterCount, &totalCount); err != nil {
		return nil, 0, 0, nil, nil, fmt.Errorf("error counting logs: %v", err)
	}

	// Data query: filters plus the cursor condition.
	var args []any
	whereClause := buildWhereClause(filters, cursor, sortField, sortOrder, direction, &args)

	queryBuilder := strings.Builder{}
	queryBuilder.WriteString("SELECT rowid, facility, severity, timestamp, hostname, app_name, procid, msgid, structured_data, msg FROM logs")
	if whereClause != "" {
		queryBuilder.WriteString(" WHERE " + whereClause)
	}
	queryBuilder.WriteString(fmt.Sprintf(" ORDER BY %s %s, rowid %s", column, sortOrder, sortOrder))
	queryBuilder.WriteString(fmt.Sprintf(" LIMIT %d", limit))

	rows, err := db.Query(queryBuilder.String(), args...)
	if err != nil {
		return nil, 0, 0, nil, nil, fmt.Errorf("error querying logs: %v", err)
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
			return nil, 0, 0, nil, nil, fmt.Errorf("error scanning log row: %v", err)
		}

		// Parse timestamp
		entry.Timestamp, err = time.Parse(time.RFC3339Nano, timestampStr)
		if err != nil {
			return nil, 0, 0, nil, nil, fmt.Errorf("error parsing timestamp: %v", err)
		}

		logs = append(logs, entry)
	}

	// Cursors point at the boundary rows of the returned page.
	var nextCursor, prevCursor *string
	if len(logs) > 0 {
		first := formatCursor(cursorValue(logs[0], sortField), logs[0].RowID)
		last := formatCursor(cursorValue(logs[len(logs)-1], sortField), logs[len(logs)-1].RowID)
		prevCursor, nextCursor = &first, &last
	}

	return logs, totalCount, filterCount, nextCursor, prevCursor, nil
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

		whereClause := buildWhereClause(facetFilters, nil, "", "", "", &args)
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

		whereClause := buildWhereClause(facetFilters, nil, "", "", "", &args)
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
func GetChartData(now time.Time, filters map[string]any) ([]ChartDataPoint, error) {
	chartFilters := make(map[string]any)
	for k, v := range filters {
		chartFilters[k] = v
	}

	// If date filters are not provided, we use the current time as the end
	// time for chart data and go back 24 hours to get the last 24 hours of data.
	if chartFilters["startDate"] == nil || chartFilters["endDate"] == nil {
		endDate := now.Truncate(time.Hour).Add(time.Hour)
		startDate := now.Add(-24 * time.Hour)
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
	whereClause := buildWhereClause(chartFilters, nil, "", "", "", &args)
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

// buildWhereClause builds the WHERE clause for the given filters and, when
// cursor is not nil, the keyset comparison for the current page.
func buildWhereClause(filters map[string]any, cursor *Cursor, sortField, sortOrder, direction string, args *[]any) string {
	if len(filters) == 0 && cursor == nil {
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

	if cursor != nil {
		conditions = append(conditions, buildCursorCondition(sortField, logSortColumns[sortField], sortOrder, direction, *cursor, args))
	}

	return strings.Join(conditions, " AND ")
}

// buildCursorCondition builds the keyset comparison for the given cursor.
//
// direction "next" moves towards later pages in the current sort order, "prev"
// towards earlier ones. rowid is used to break ties on the sort column.
func buildCursorCondition(sortField, column, sortOrder, direction string, cursor Cursor, args *[]any) string {
	forward := direction != "prev"
	ascending := sortOrder == "ASC"

	op := "<"
	if ascending == forward {
		op = ">"
	}

	*args = append(*args, cursorArg(sortField, cursor.Value), cursorArg(sortField, cursor.Value), cursor.RowID)
	return fmt.Sprintf("(%s %s ? OR (%s = ? AND rowid %s ?))", column, op, column, op)
}
