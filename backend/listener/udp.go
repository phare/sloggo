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

	"github.com/leodido/go-syslog/v4"
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
	// Process the input using go-syslog parser
	input := string(message)

	// For UDP, we need to handle each datagram separately
	// Split by newlines in case multiple messages were sent in one datagram
	parts := strings.SplitSeq(strings.ReplaceAll(input, "\r\n", "\n"), "\n")

	// Create a parser with best effort mode when RFC5424 is enabled
	var parser syslog.Machine
	if utils.LogFormat == "rfc5424" || utils.LogFormat == "auto" {
		parser = rfc5424.NewParser(rfc5424.WithBestEffort())
	}

	for part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue // Skip empty messages
		}

		parsed := false
		var lastErr error

		// Try RFC5424 if enabled
		if parser != nil && (utils.LogFormat == "rfc5424" || utils.LogFormat == "auto") {
			if syslogMsg, err := parser.Parse([]byte(part)); err == nil {
				if rfc5424Msg, ok := syslogMsg.(*rfc5424.SyslogMessage); ok {
					if logEntry := formats.SyslogMessageToLogEntry(rfc5424Msg); logEntry != nil {
						if err := db.StoreLog(*logEntry); err != nil {
							log.Printf("Error storing UDP log: %v", err)
						}
						parsed = true
					}
				}
			} else {
				lastErr = err
			}
		}

		// Try RFC3164 if enabled and not yet parsed
		if !parsed && (utils.LogFormat == "rfc3164" || utils.LogFormat == "auto") {
			if logEntry, err := formats.ParseRFC3164ToLogEntry(part); err == nil {
				if err := db.StoreLog(*logEntry); err != nil {
					log.Printf("Error storing UDP log: %v", err)
				}
				parsed = true
			} else {
				lastErr = err
			}
		}

		if !parsed {
			log.Printf("Failed to parse UDP message with format %s: %v: %s", utils.LogFormat, lastErr, input)
		}
	}
}
