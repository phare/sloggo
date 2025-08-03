package db

import (
	"database/sql"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

var (
	dbDirectory string
	dbInstance  *sql.DB
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
}

// GetDBInstance returns the initialized SQLite database instance.
func GetDBInstance() *sql.DB {
	return dbInstance
}

// StoreLog stores an RFC5424 log message in the SQLite database.
func StoreLog(query string, params []any) error {
	_, err := dbInstance.Exec(query, params...)

	if err != nil {
		log.Printf("Failed to store log in database: %v", err)
		return err
	}

	return nil
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

	dbInstance, err = sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_synchronous=OFF&_cache_size=-100000")
	if err != nil {
		log.Fatalf("Failed to connect to SQLite database: %v", err)
	}
}
