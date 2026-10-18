package db

import (
	"fmt"
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

// TestGetLogsPaginationWithDuplicateTimestamps verifies that keyset pagination
// returns every row exactly once when all rows share the same timestamp.
func TestGetLogsPaginationWithDuplicateTimestamps(t *testing.T) {
	const (
		hostname = "pagination-dup-host"
		total    = 25
		pageSize = 10
	)
	timestamp := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)

	for i := 0; i < total; i++ {
		err := StoreLog(models.LogEntry{
			Severity:       6,
			Facility:       1,
			Version:        1,
			Timestamp:      timestamp,
			Hostname:       hostname,
			AppName:        "pagination-app",
			ProcID:         "-",
			MsgID:          "-",
			StructuredData: "-",
			Message:        fmt.Sprintf("pagination message %d", i),
		})
		if err != nil {
			t.Fatalf("Failed to store log entry: %v", err)
		}
	}
	if err := ProcessBatchStoreLogs(); err != nil {
		t.Fatalf("Failed to process batch: %v", err)
	}

	filters := map[string]any{"hostname": hostname}

	for _, sortOrder := range []string{"DESC", "ASC"} {
		t.Run(sortOrder, func(t *testing.T) {
			seen := make(map[int64]bool, total)
			var cursor *Cursor
			pages := 0

			for {
				logs, _, filterCount, nextCursor, _, err := GetLogs(pageSize, cursor, "next", filters, "timestamp", sortOrder)
				if err != nil {
					t.Fatalf("GetLogs failed: %v", err)
				}
				if filterCount != total {
					t.Fatalf("filterCount = %d, want %d", filterCount, total)
				}
				for _, entry := range logs {
					if seen[entry.RowID] {
						t.Fatalf("row %d returned more than once", entry.RowID)
					}
					seen[entry.RowID] = true
				}

				pages++
				if nextCursor == nil {
					break
				}
				parsed, ok := ParseCursor(*nextCursor)
				if !ok {
					t.Fatalf("invalid cursor %q", *nextCursor)
				}
				cursor = &parsed

				if pages > total {
					t.Fatal("pagination did not terminate")
				}
			}

			if len(seen) != total {
				t.Fatalf("returned %d unique rows, want %d", len(seen), total)
			}
		})
	}
}
