package listener

import (
	"log"
	"net"
	"os"
	"sloggo/db"
	"sloggo/formats"
	"time"
)

// StartTCPListener starts a TCP listener on port 6514.
// Logs received are parsed and stored in the SQLite database.
func StartTCPListener() {
	port := os.Getenv("TCP_PORT")

	if port == "" {
		port = "6514"
	}

	_, err := net.LookupPort("tcp", port)
	if err != nil {
		log.Fatalf("Invalid TCP port %s: %v", port, err)
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to start TCP listener on port %s: %v", port, err)
	}
	defer listener.Close()

	log.Printf("TCP listener is running on port :%s", port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting TCP connection: %v", err)
			continue
		}
		go handleTCPConnection(conn)
	}
}

func handleTCPConnection(conn net.Conn) {
	defer conn.Close()

	// Set read deadline to prevent hanging
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		log.Printf("Error setting read deadline: %v", err)
		return
	}

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			log.Printf("Connection read timed out")
		} else {
			log.Printf("Error reading from TCP connection: %v", err)
		}
		return
	}

	message := string(buffer[:n])

	// Parse the log message as RFC5424
	logEntry, err := formats.NewRFC5424Log(message)
	if err != nil {
		log.Printf("Failed to parse log message: %v", err)
		return
	}

	query, params := logEntry.ToSQL()

	// Store the log in the database
	if err := db.StoreLog(query, params); err != nil {
		log.Printf("Failed to store log message: %v", err)
		return
	}
}
