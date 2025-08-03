package db

import (
	"database/sql"
	"log"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"
	"time"

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
)

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
