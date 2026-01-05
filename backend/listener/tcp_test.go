package listener

import (
	"fmt"
	"net"
	"sloggo/db"
	"sloggo/utils"
	"strings"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb/v2"
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
	// Allow the message to be processed - longer wait time
	time.Sleep(100 * time.Millisecond)
}

func TestTCPListener(t *testing.T) {
	// Save original LogFormat and restore at the end
	originalLogFormat := utils.GetLogFormat()
	defer func() {
		utils.SetLogFormat(originalLogFormat)
	}()

	// Clean the database before starting tests
	db := db.GetDBInstance()
	_, err := db.Exec("DELETE FROM logs")
	if err != nil {
		t.Fatalf("Failed to clean database: %v", err)
	}

	checkSchema(t)

	port := 6514
	done := make(chan bool) // Channel to signal listener is running

	// Start TCP listener in a goroutine
	go func() {
		// Signal that we're about to start
		done <- true
		StartTCPListener()
	}()

	// Wait for signal that listener is starting
	<-done

	// Allow the listener to fully initialize
	time.Sleep(2 * time.Second)

	testCases := getTestCases()

	// Create a single connection for all test cases
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Fatalf("Failed to connect to TCP listener: %v", err)
	}
	defer conn.Close()

	// Run test cases sequentially on the same connection for different log formats
	formats := []string{"auto", "rfc5424", "rfc3164"}
	for _, format := range formats {
		// Set format for this test group using thread-safe function
		utils.SetLogFormat(format)

		for _, tc := range testCases {
			testName := fmt.Sprintf("%s_%s", format, tc.name)
			t.Logf("Running test: %s", testName)
			sendTCPMessage(t, conn, tc.message)
			// No need to explicitly force batch processing - handled in verifyLogEntry
			verifyLogEntry(t, tc)
		}
	}
}
