package listener

import (
	"bufio"
	"log"
	"net"
	"sloggo/db"
	"sloggo/formats"
	"sloggo/utils"
	"time"

	"github.com/leodido/go-syslog/v4/rfc5424"
)

func StartTCPListener() {
	port := utils.TcpPort

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

// handleTCPConnection handles a TCP connection
func handleTCPConnection(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	parser := rfc5424.NewParser(rfc5424.WithBestEffort())

	// Configure scanner with a larger buffer for bigger messages
	const maxScanSize = 1024 * 1024 // 1MB max message size
	buffer := make([]byte, 0, 64*1024)
	scanner.Buffer(buffer, maxScanSize)

	// Use the default ScanLines function which already handles newlines
	// This simplifies our code and handles both \n and \r\n properly

	// Set initial read deadline
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	for {
		// Scan for the next message
		if !scanner.Scan() {
			// Check for errors
			if err := scanner.Err(); err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// Just a timeout, reset deadline and try again
					conn.SetReadDeadline(time.Now().Add(10 * time.Second))
					continue
				}
				log.Printf("TCP connection closed: %v", err)
			}
			// EOF or error occurred
			return
		}

		// Reset deadline after successful read
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))

		// Get the message text
		message := scanner.Text()
		if message == "" {
			// Skip empty messages (like keepalives)
			continue
		}

		// Parse the message
		syslogMsg, err := parser.Parse([]byte(message))
		if err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		// Convert to RFC5424 syslog message
		rfc5424Msg, ok := syslogMsg.(*rfc5424.SyslogMessage)
		if !ok {
			log.Printf("Parsed message is not a valid RFC5424 message")
			continue
		}

		// Convert directly to SQL without intermediate format
		query, params := formats.SyslogMessageToSQL(rfc5424Msg)
		db.StoreLog(query, params)
	}
}
