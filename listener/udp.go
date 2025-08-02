package listener

import (
	"database/sql"
	"log"
	"net"
	"os"
)

// StartUDPListener starts a UDP listener on the specified port.
// The port can be configured using the environment variable "UDP_PORT".
// Logs received are stored in the provided SQLite database.
func StartUDPListener(db *sql.DB) {
	// Get the port from the environment variable or use the default.
	port := os.Getenv("UDP_PORT")
	if port == "" {
		port = "514" // Default UDP port
	}

	addr := net.UDPAddr{
		Port: parsePort(port),
		IP:   net.ParseIP("0.0.0.0"),
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalf("Failed to start UDP listener: %v", err)
	}
	defer conn.Close()

	log.Printf("UDP listener is running on :%s", port)

	buffer := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error reading from UDP: %v", err)
			continue
		}
		message := string(buffer[:n])
		storeLog(db, message)
	}
}

// parsePort converts a port string to an integer and handles errors.
func parsePort(port string) int {
	parsedPort, err := net.LookupPort("udp", port)
	if err != nil {
		log.Fatalf("Invalid UDP port: %v", err)
	}
	return parsedPort
}

// storeLog inserts a log message into the SQLite database.
func storeLog(db *sql.DB, message string) {
	_, err := db.Exec("INSERT INTO logs (message) VALUES (?)", message)
	if err != nil {
		log.Printf("Failed to store log in database: %v", err)
	} else {
		log.Println("Log stored successfully")
	}
}
