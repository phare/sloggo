package listener

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"sloggo/db"
)

// StartTCPListener starts a TCP listener on port 6514.
// Logs received are parsed and stored in the SQLite database.
func StartTCPListener(dbConn *sql.DB) {
	port := 6514
	address := ":" + fmt.Sprintf("%d", port)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to start TCP listener on port %d: %v", port, err)
	}
	defer listener.Close()

	log.Printf("TCP listener is running on port %d", port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting TCP connection: %v", err)
			continue
		}
		go handleTCPConnection(conn, dbConn)
	}
}

func handleTCPConnection(conn net.Conn, dbConn *sql.DB) {
	defer conn.Close()
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Printf("Error reading from TCP connection: %v", err)
		return
	}
	message := string(buffer[:n])

	// Example parsing logic for RFC 5424 (simplified)
	hostname := "example-host"
	appName := "example-app"
	procID := "1234"
	msgID := "5678"

	// Store the log in the database
	db.StoreLog(dbConn, hostname, appName, procID, msgID, message)
}
