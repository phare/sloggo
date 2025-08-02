package main

import (
	"database/sql"
	"log"
	"sloggo/server"

	"sloggo/db"
	"sloggo/listener"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Initialize SQLite database
	dbConn, err := sql.Open("sqlite3", "./logs.db")

	if err != nil {
		log.Fatal("Failed to connect to SQLite database:", err)
	}
	defer dbConn.Close()

	// Initialize database schema
	if err := db.InitializeSchema(dbConn); err != nil {
		log.Fatal("Failed to initialize database schema:", err)
	}

	// Start UDP listener
	go listener.StartUDPListener(dbConn)

	// Start TCP listener
	go listener.StartTCPListener(dbConn)

	// Start HTTP server
	server.StartHTTPServer()
}
