package listener

import (
	"fmt"
	"net"
	"sloggo/utils"
	"strings"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb/v2"
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

	formats := []string{"auto", "rfc5424", "rfc3164"}
	for _, format := range formats {
		utils.LogFormat = format
		for _, tc := range testCases {
			name := fmt.Sprintf("%s_%s", format, tc.name)
			t.Run(name, func(t *testing.T) {
				sendUDPMessage(t, fmt.Sprintf("localhost:%d", port), tc.message)
				verifyLogEntry(t, tc)
			})
		}
	}
}
