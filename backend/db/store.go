package db

import (
	"database/sql"
	"log"
	"os"
	"path"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	dbDirectory string
	dbInstance  *sql.DB
	once        sync.Once
)

func init() {
	e, err := os.Executable()

	if err != nil {
		log.Fatal(err)
	}

	dbDirectory = filepath.Join(path.Dir(e), "/.sqlite")

	log.Printf("DB DIRECTORY", dbDirectory)

	dbInstance, err = sql.Open("sqlite3", filepath.Join(dbDirectory, "logs.db"))
	if err != nil {
		log.Fatalf("Failed to connect to SQLite database: %v", err)
	}

	// Initialize schema
	query := `
	CREATE TABLE IF NOT EXISTS logs (
	    id INTEGER PRIMARY KEY AUTOINCREMENT,
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

	if _, err = dbInstance.Exec(query); err != nil {
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
