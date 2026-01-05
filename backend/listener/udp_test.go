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
	// Save original LogFormat and restore at the end
	originalLogFormat := utils.GetLogFormat()
	defer func() {
		utils.SetLogFormat(originalLogFormat)
	}()

	checkSchema(t)

	port := 5514
	go StartUDPListener()

	// Allow the listener to start
	time.Sleep(1 * time.Second)

	testCases := getTestCases()

	// Run test cases sequentially for different log formats
	// We must test formats sequentially to avoid race conditions on utils.LogFormat
	// Note: Not using nested t.Run() to ensure truly serial execution
	formats := []string{"auto", "rfc5424", "rfc3164"}
	for _, format := range formats {
		// Set format for this test group using thread-safe function
		utils.SetLogFormat(format)

		for _, tc := range testCases {
			testName := fmt.Sprintf("%s_%s", format, tc.name)
			t.Logf("Running test: %s", testName)
			sendUDPMessage(t, fmt.Sprintf("localhost:%d", port), tc.message)
			verifyLogEntry(t, tc)
		}
	}
}
