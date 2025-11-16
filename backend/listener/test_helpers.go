package listener

import (
	"sloggo/db"
	"strings"
	"testing"
	"time"
)

type expectedResult struct {
	facility       int
	severity       int
	hostname       string
	appName        string
	procid         string
	msgid          string
	structuredData string
	msg            string
	shouldError    bool
}

type testCase struct {
	name     string
	message  string
	expected expectedResult
}

func checkSchema(t *testing.T) {
	db := db.GetDBInstance()
	rows, err := db.Query(`
		SELECT sql FROM sqlite_master
		WHERE type='table' AND name='logs'
	`)
	if err != nil {
		t.Fatalf("Failed to query schema: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("Logs table not found in database")
	}

	var createSQL string
	if err := rows.Scan(&createSQL); err != nil {
		t.Fatalf("Failed to scan create SQL: %v", err)
	}
}

func verifyLogEntry(t *testing.T, tc testCase) {
	var hostname, appName, procid, msgid, message, structuredData string
	var severity, facility int

	err := db.ProcessBatchStoreLogs()
	if err != nil {
		t.Fatalf("Failed to process batch: %v", err)
	}

	// Wait a bit more for any database operations to complete
	time.Sleep(200 * time.Millisecond)

	// First check what's in the database for debugging
	rows, err := db.GetDBInstance().Query("SELECT hostname, app_name, procid, msgid, msg, structured_data, severity, facility FROM logs")
	if err != nil {
		t.Fatalf("Failed to query database for debug: %v", err)
	}
	defer rows.Close()

	found := false
	for rows.Next() {
		var h, a, p, m, msg, sd string
		var sev, fac int
		if err := rows.Scan(&h, &a, &p, &m, &msg, &sd, &sev, &fac); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}

		if strings.TrimSpace(msg) == strings.TrimSpace(tc.expected.msg) {
			hostname, appName, procid, msgid = h, a, p, m
			message, structuredData = msg, sd
			severity, facility = sev, fac
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Message not found in database")
	}

	if tc.expected.shouldError {
		if err == nil {
			t.Error("Expected error but got none")
		}
		return
	}

	if hostname != tc.expected.hostname ||
		appName != tc.expected.appName ||
		procid != tc.expected.procid ||
		msgid != tc.expected.msgid ||
		severity != tc.expected.severity ||
		facility != tc.expected.facility ||
		message != tc.expected.msg ||
		structuredData != tc.expected.structuredData {
		t.Errorf(`Log message fields do not match:
    Hostname:        got=%q want=%q
    AppName:         got=%q want=%q
    ProcID:          got=%q want=%q
    MsgID:           got=%q want=%q
    Message:         got=%q want=%q
    StructuredData:  got=%q want=%q
    Severity:        got=%d want=%d
    Facility:        got=%d want=%d`,
			hostname, tc.expected.hostname,
			appName, tc.expected.appName,
			procid, tc.expected.procid,
			msgid, tc.expected.msgid,
			message, tc.expected.msg,
			structuredData, tc.expected.structuredData,
			severity, tc.expected.severity,
			facility, tc.expected.facility)
	}
}

func getTestCases() []testCase {
	return []testCase{
		{
			name:    "Valid message with basic fields",
			message: "<13>1 2023-10-01T12:34:56Z example-host example-app 1234 5678 - Test log message",
			expected: expectedResult{
				facility:       1,
				severity:       5,
				hostname:       "example-host",
				appName:        "example-app",
				procid:         "1234",
				msgid:          "5678",
				structuredData: "-",
				msg:            "Test log message",
				shouldError:    false,
			},
		},
		{
			name:    "Message with structured data",
			message: "<165>1 2023-10-01T12:34:56Z host1 app1 2345 ID01 [exampleSDID@32473 iut=\"3\" eventSource=\"Application\"] Message with structured data",
			expected: expectedResult{
				facility:       20,
				severity:       5,
				hostname:       "host1",
				appName:        "app1",
				procid:         "2345",
				msgid:          "ID01",
				structuredData: "{\"exampleSDID@32473\":{\"eventSource\":\"Application\",\"iut\":\"3\"}}",
				msg:            "Message with structured data",
				shouldError:    false,
			},
		},
		{
			name:    "Emergency message from kernel",
			message: "<0>1 2023-10-01T12:34:56Z host2 kernel 0 - - Kernel panic - not syncing",
			expected: expectedResult{
				facility:       0,
				severity:       0,
				hostname:       "host2",
				appName:        "kernel",
				procid:         "0",
				msgid:          "-",
				structuredData: "-",
				msg:            "Kernel panic - not syncing",
				shouldError:    false,
			},
		},
		{
			name:    "RFC3164 basic without pid",
			message: "<34>Oct 11 22:14:15 mymachine su: 'su root' failed for lonvick on /dev/pts/8",
			expected: expectedResult{
				facility:       4,
				severity:       2,
				hostname:       "mymachine",
				appName:        "su",
				procid:         "-",
				msgid:          "-",
				structuredData: "-",
				msg:            "'su root' failed for lonvick on /dev/pts/8",
				shouldError:    false,
			},
		},
		{
			name:    "RFC3164 with pid typical esphome",
			message: "<190>Nov  6 09:01:02 esphome-device esphome[1234]: Sensor reading: 42",
			expected: expectedResult{
				facility:       23,
				severity:       6,
				hostname:       "esphome-device",
				appName:        "esphome",
				procid:         "1234",
				msgid:          "-",
				structuredData: "-",
				msg:            "Sensor reading: 42",
				shouldError:    false,
			},
		},
	}
}
