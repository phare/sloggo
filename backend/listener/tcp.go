package listener

import (
	"bufio"
	"io"
	"log"
	"net"
	"strconv"
	"sloggo/db"
	"sloggo/formats"
	"sloggo/utils"
	"strings"
	"sync"
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

	// Use a semaphore to limit concurrent processors
	maxConcurrentProcessors := 100
	semaphore := make(chan struct{}, maxConcurrentProcessors)

	// Create a WaitGroup to track active connections
	var wg sync.WaitGroup

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting TCP connection: %v", err)
			continue
		}

		select {
		case semaphore <- struct{}{}:
			// Slot acquired, process the connection
			wg.Add(1)

			go func(c net.Conn) {
				defer func() {
					// Release resources when done
					<-semaphore
					wg.Done()
				}()
				handleTCPConnection(c)
			}(conn)
		default:
			log.Printf("Warning: TCP connection processing at capacity, rejecting connection")
			conn.Close()
		}
	}
}

// handleTCPConnection handles a TCP connection
func handleTCPConnection(conn net.Conn) {
	defer conn.Close()

	// Set up TCP keep-alive to maintain connection
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	// Create a buffered reader to handle both octet counting and newline-delimited formats
	reader := bufio.NewReader(conn)
	parser := rfc5424.NewParser(rfc5424.WithBestEffort())

	for {
		// Read message in either octet counting format (RFC 6587) or newline-delimited
		message, err := readSyslogMessage(reader)
		if err != nil {
			if err.Error() == "EOF" {
				log.Printf("TCP connection closed by client")
			} else {
				log.Printf("Error reading message: %v", err)
			}
			return
		}

		message = strings.TrimSpace(message)
		if message == "" {
			continue
		}

		// Parse the message
		syslogMsg, err := parser.Parse([]byte(message))
		if err != nil {
			log.Printf("Failed to parse message: %v: %s", err, message)
			continue
		}

		// Convert to RFC5424 syslog message
		rfc5424Msg, ok := syslogMsg.(*rfc5424.SyslogMessage)
		if !ok {
			log.Printf("Parsed message is not a valid RFC5424 message: %s", message)
			continue
		}

		// Convert directly to LogEntry for efficient DuckDB insertion
		logEntry := formats.SyslogMessageToLogEntry(rfc5424Msg)

		if logEntry == nil {
			log.Printf("Failed to convert message to LogEntry: %s", message)
		}

		// Store log without blocking if possible
		if err := db.StoreLog(*logEntry); err != nil {
			log.Printf("Error storing log: %v", err)
		}
	}
}

// readSyslogMessage reads a syslog message in either octet counting or newline-delimited format
func readSyslogMessage(reader *bufio.Reader) (string, error) {
	// Peek at the first few bytes to determine the format
	peekBytes, err := reader.Peek(10)
	if err != nil {
		return "", err
	}

	// Check if the message starts with a digit (octet counting format: "length message")
	if len(peekBytes) > 0 && peekBytes[0] >= '0' && peekBytes[0] <= '9' {
		// Parse the length prefix in octet counting format
		return readOctetCountingMessage(reader)
	} else {
		// Use newline-delimited format
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		// Remove the newline character
		return strings.TrimSuffix(line, "\n"), nil
	}
}

// readOctetCountingMessage reads a message in octet counting format (RFC 6587)
func readOctetCountingMessage(reader *bufio.Reader) (string, error) {
	// Read the length prefix (digits followed by a space)
	var lengthStr string
	for {
		char, err := reader.ReadByte()
		if err != nil {
			return "", err
		}
		
		if char == ' ' {
			break // End of length prefix
		}
		
		if char < '0' || char > '9' {
			// Not a valid octet counting format, put back the character and return as-is
			reader.UnreadByte()
			return lengthStr, nil
		}
		
		lengthStr += string(char)
	}

	// Convert length string to integer
	msgLen, err := strconv.Atoi(lengthStr)
	if err != nil {
		return "", err
	}

	// Read exactly msgLen bytes
	message := make([]byte, msgLen)
	_, err = io.ReadFull(reader, message)
	if err != nil {
		return "", err
	}

	return string(message), nil
}
