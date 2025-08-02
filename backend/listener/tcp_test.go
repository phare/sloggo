package listener

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func sendTCPMessage(t *testing.T, conn net.Conn, message string) {
	// Ensure message ends with newline
	if !strings.HasSuffix(message, "\n") {
		message += "\n"
	}
	_, err := conn.Write([]byte(message))
	if err != nil {
		t.Fatalf("Failed to send log message: %v", err)
	}
	// Allow the message to be processed
	time.Sleep(100 * time.Millisecond)
}

func TestTCPListener(t *testing.T) {
	checkSchema(t)

	port := 6514
	go StartTCPListener()

	// Allow the listener to start
	time.Sleep(1 * time.Second)

	testCases := getTestCases()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
			if err != nil {
				t.Fatalf("Failed to connect to TCP listener: %v", err)
			}
			defer conn.Close()

			sendTCPMessage(t, conn, tc.message)
			verifyLogEntry(t, tc)
		})
	}
}
