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

	listener, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalf("Failed to start UDP listener on port %s: %v", port, err)
	}
	defer listener.Close()

	log.Printf("UDP listener is running on port :%s", port)

	// Use a semaphore to limit concurrent processors
	maxConcurrentProcessors := 100
	semaphore := make(chan struct{}, maxConcurrentProcessors)

	// Use a WaitGroup to track active processors
	var wg sync.WaitGroup

	// Configure a larger buffer for UDP packets
	const bufferSize = 64 * 1024 // 64KB buffer
	buffer := make([]byte, bufferSize)

	for {
		listener.SetReadDeadline(time.Now().Add(30 * time.Second))

		n, _, err := listener.ReadFromUDP(buffer)
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
			log.Printf("Warning: UDP connection processing at capacity, rejecting connection")
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
	parts := strings.SplitSeq(strings.ReplaceAll(input, "\r\n", "\n"), "\n")

	for part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue // Skip empty messages
		}

		// Parse the message
		syslogMsg, err := parser.Parse([]byte(part))
		if err != nil {
			log.Printf("Failed to parse UDP message: %v: %s", err, input)
			continue
		}

		// Convert to RFC5424 syslog message
		rfc5424Msg, ok := syslogMsg.(*rfc5424.SyslogMessage)
		if !ok {
			log.Printf("Parsed UDP message is not a valid RFC5424 message: %s", input)
			continue
		}

		// Convert directly to LogEntry for efficient DuckDB insertion
		logEntry := formats.SyslogMessageToLogEntry(rfc5424Msg)

		if logEntry == nil {
			log.Printf("Failed to convert message to LogEntry: %s", message)
		}

		// Store log without blocking if possible
		if err := db.StoreLog(*logEntry); err != nil {
			log.Printf("Error storing UDP log: %v", err)
		}
	}
}
