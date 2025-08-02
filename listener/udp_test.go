package listener

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func sendUDPMessage(t *testing.T, addr string, message string) {
	// Ensure message ends with newline
	if !strings.HasSuffix(message, "\n") {
		message += "\n"
	}
	conn, err := net.Dial("udp", addr)
	if err != nil {
		t.Fatalf("Failed to create UDP connection: %v", err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(message))
	if err != nil {
		t.Fatalf("Failed to send UDP message: %v", err)
	}
	// Allow the message to be processed
	time.Sleep(100 * time.Millisecond)
}

func TestUDPListener(t *testing.T) {
	checkSchema(t)

	port := 5514
	go StartUDPListener()

	// Allow the listener to start
	time.Sleep(1 * time.Second)

	testCases := getTestCases()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sendUDPMessage(t, fmt.Sprintf("localhost:%d", port), tc.message)
			verifyLogEntry(t, tc)
		})
	}
}
