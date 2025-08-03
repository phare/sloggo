package listener

import (
	"log"
	"net"
	"sloggo/db"
	"sloggo/formats"
	"sloggo/utils"
	"strings"
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

	// Configure a larger buffer for UDP packets
	const bufferSize = 64 * 1024 // 64KB buffer
	buffer := make([]byte, bufferSize)

	// Create a parser with best effort mode
	parser := rfc5424.NewParser(rfc5424.WithBestEffort())

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

		// Process the input using go-syslog parser
		input := string(buffer[:n])

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
			db.StoreLog(query, params)
		}
	}
}
