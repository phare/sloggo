package listener

import (
	"log"
	"net"
	"sloggo/db"
	"sloggo/formats"
	"sloggo/utils"
	"strings"
	"sync"
	"time"

	"github.com/leodido/go-syslog/v4/rfc5424"
)

func StartUDPListener() {
	port := utils.UdpPort

	intPort, err := net.LookupPort("udp", port)
	if err != nil {
		log.Fatalf("Invalid UDP port %s: %v", port, err)
	}

	addr := net.UDPAddr{
		Port: intPort,
		IP:   net.ParseIP("0.0.0.0"),
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalf("Failed to start UDP listener on port %s: %v", port, err)
	}
	defer conn.Close()

	log.Printf("UDP listener is running on port :%s", port)

	// Use a semaphore to limit concurrent message processing
	maxConcurrentProcessors := 100
	semaphore := make(chan struct{}, maxConcurrentProcessors)

	// Use a WaitGroup to track active processors
	var wg sync.WaitGroup

	// Configure a larger buffer for UDP packets
	const bufferSize = 64 * 1024 // 64KB buffer
	buffer := make([]byte, bufferSize)

	for {
		// Set read deadline for UDP socket
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))

		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Just a timeout, continue
				continue
			}
			log.Printf("Error reading from UDP: %v", err)
			continue
		}

		// Make a copy of the received data to process
		messageCopy := make([]byte, n)
		copy(messageCopy, buffer[:n])

		// Acquire semaphore slot (non-blocking)
		select {
		case semaphore <- struct{}{}:
			// Slot acquired, process the message
			wg.Add(1)
			go func(data []byte) {
				defer func() {
					// Release resources when done
					<-semaphore
					wg.Done()
				}()
				processUDPMessage(data)
			}(messageCopy)
		default:
			// Semaphore full, log a warning and continue
			log.Printf("Warning: UDP message processing at capacity, dropping message")
		}
	}
}

// processUDPMessage handles processing of a single UDP message
func processUDPMessage(message []byte) {
	// Create a parser with best effort mode
	parser := rfc5424.NewParser(rfc5424.WithBestEffort())

	// Process the input using go-syslog parser
	input := string(message)

	// For UDP, we need to handle each datagram separately
	// Split by newlines in case multiple messages were sent in one datagram
	parts := strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue // Skip empty messages
		}

		// Parse the message
		syslogMsg, err := parser.Parse([]byte(part))
		if err != nil {
			log.Printf("Failed to parse UDP message: %v", err)
			continue
		}

		// Convert to RFC5424 syslog message
		rfc5424Msg, ok := syslogMsg.(*rfc5424.SyslogMessage)
		if !ok {
			log.Printf("Parsed UDP message is not a valid RFC5424 message")
			continue
		}

		// Convert directly to SQL without intermediate format
		query, params := formats.SyslogMessageToSQL(rfc5424Msg)

		// Store log without blocking if possible
		if err := db.StoreLog(query, params); err != nil {
			log.Printf("Error storing UDP log: %v", err)
		}
	}
}
