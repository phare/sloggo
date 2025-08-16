package db

import (
	"sloggo/models"
	"testing"
	"time"
)

func TestStoreLogEntry(t *testing.T) {
	entry := models.LogEntry{
		Severity:       5,
		Facility:       1,
		Version:        1,
		Timestamp:      time.Now(),
		Hostname:       "test-host",
		AppName:        "test-app",
		ProcID:         "1234",
		MsgID:          "5678",
		StructuredData: "-",
		Message:        "Test message",
	}

	err := StoreLog(entry)
	if err != nil {
		t.Fatalf("Failed to store log entry: %v", err)
	}

	err = ProcessBatchStoreLogs()
	if err != nil {
		t.Fatalf("Failed to process batch: %v", err)
	}

	//	time.Sleep(100 * time.Millisecond)

	db := GetDBInstance()
	rows, err := db.Query(`
		SELECT severity, facility, version, hostname, app_name, procid, msgid, structured_data, msg
		FROM logs
		WHERE hostname = ? AND app_name = ? AND msg = ?
	`, entry.Hostname, entry.AppName, entry.Message)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("Expected log entry not found in database")
	}

	var severity, facility uint8
	var version uint16
	var hostname, appName, procID, msgID, structuredData, message string

	err = rows.Scan(&severity, &facility, &version, &hostname, &appName, &procID, &msgID, &structuredData, &message)
	if err != nil {
		t.Fatalf("Failed to scan row: %v", err)
	}

	// Verify all fields match
	if severity != entry.Severity {
		t.Errorf("Severity: got %d, want %d", severity, entry.Severity)
	}
	if facility != entry.Facility {
		t.Errorf("Facility: got %d, want %d", facility, entry.Facility)
	}
	if version != entry.Version {
		t.Errorf("Version: got %d, want %d", version, entry.Version)
	}
	if hostname != entry.Hostname {
		t.Errorf("Hostname: got %q, want %q", hostname, entry.Hostname)
	}
	if appName != entry.AppName {
		t.Errorf("AppName: got %q, want %q", appName, entry.AppName)
	}
	if procID != entry.ProcID {
		t.Errorf("ProcID: got %q, want %q", procID, entry.ProcID)
	}
	if msgID != entry.MsgID {
		t.Errorf("MsgID: got %q, want %q", msgID, entry.MsgID)
	}
	if structuredData != entry.StructuredData {
		t.Errorf("StructuredData: got %q, want %q", structuredData, entry.StructuredData)
	}
	if message != entry.Message {
		t.Errorf("Message: got %q, want %q", message, entry.Message)
	}
}

func TestBatchProcessing(t *testing.T) {
	entries := []models.LogEntry{
		{
			Severity:       3,
			Facility:       2,
			Version:        1,
			Timestamp:      time.Now(),
			Hostname:       "batch-host-1",
			AppName:        "batch-app",
			ProcID:         "100",
			MsgID:          "MSG1",
			StructuredData: "-",
			Message:        "Batch message 1",
		},
		{
			Severity:       4,
			Facility:       3,
			Version:        1,
			Timestamp:      time.Now(),
			Hostname:       "batch-host-2",
			AppName:        "batch-app",
			ProcID:         "200",
			MsgID:          "MSG2",
			StructuredData: "-",
			Message:        "Batch message 2",
		},
		{
			Severity:       5,
			Facility:       4,
			Version:        1,
			Timestamp:      time.Now(),
			Hostname:       "batch-host-3",
			AppName:        "batch-app",
			ProcID:         "300",
			MsgID:          "MSG3",
			StructuredData: "-",
			Message:        "Batch message 3",
		},
	}

	for _, entry := range entries {
		err := StoreLog(entry)
		if err != nil {
			t.Fatalf("Failed to store log entry: %v", err)
		}
	}

	err := ProcessBatchStoreLogs()
	if err != nil {
		t.Fatalf("Failed to process batch: %v", err)
	}

	// Wait for processing to complete
	//	time.Sleep(100 * time.Millisecond)

	// Verify all entries are in the database
	db := GetDBInstance()
	rows, err := db.Query(`
		SELECT COUNT(*) FROM logs WHERE app_name = ?
	`, "batch-app")
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("Failed to get count from database")
	}

	var count int
	err = rows.Scan(&count)
	if err != nil {
		t.Fatalf("Failed to scan count: %v", err)
	}

	if count < len(entries) {
		t.Errorf("Expected at least %d entries in database, got %d", len(entries), count)
	}
}
