package db

import (
	"database/sql"
	"log"
)

// InitializeSchema creates the SQLite schema for storing logs in RFC 5424 format.
func InitializeSchema(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		hostname TEXT,
		app_name TEXT,
		proc_id TEXT,
		msg_id TEXT,
		message TEXT
	)`
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("Failed to create logs table: %v", err)
		return err
	}
	log.Println("Logs table created or already exists")
	return nil
}

// StoreLog inserts a log message into the SQLite database in RFC 5424 format.
func StoreLog(db *sql.DB, hostname, appName, procID, msgID, message string) {
	query := `
	INSERT INTO logs (hostname, app_name, proc_id, msg_id, message)
	VALUES (?, ?, ?, ?, ?)`
	_, err := db.Exec(query, hostname, appName, procID, msgID, message)
	if err != nil {
		log.Printf("Failed to store log in database: %v", err)
	} else {
		log.Println("Log stored successfully")
	}
}
