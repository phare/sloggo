package formats

import (
	"sloggo/models"
	"testing"

	"github.com/leodido/go-syslog/v4/rfc5424"
)

func TestSyslogMessageToLogEntry(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected models.LogEntry
	}{
		{
			name:  "Basic syslog message",
			input: "<13>1 2023-10-01T12:34:56Z example-host example-app 1234 5678 - Test log message",
			expected: models.LogEntry{
				Severity:       5, // 13 % 8 = 5
				Facility:       1, // 13 / 8 = 1
				Version:        1,
				Hostname:       "example-host",
				AppName:        "example-app",
				ProcID:         "1234",
				MsgID:          "5678",
				StructuredData: "-",
				Message:        "Test log message",
			},
		},
		{
			name:  "Emergency message",
			input: "<0>1 2023-10-01T12:34:56Z host2 kernel 0 - - Kernel panic",
			expected: models.LogEntry{
				Severity:       0, // 0 % 8 = 0
				Facility:       0, // 0 / 8 = 0
				Version:        1,
				Hostname:       "host2",
				AppName:        "kernel",
				ProcID:         "0",
				MsgID:          "-",
				StructuredData: "-",
				Message:        "Kernel panic",
			},
		},
		{
			name:  "Message with structured data",
			input: "<165>1 2023-10-01T12:34:56Z host1 app1 2345 ID01 [exampleSDID@32473 iut=\"3\" eventSource=\"Application\"] Message with structured data",
			expected: models.LogEntry{
				Severity:       5,  // 165 % 8 = 5
				Facility:       20, // 165 / 8 = 20
				Version:        1,
				Hostname:       "host1",
				AppName:        "app1",
				ProcID:         "2345",
				MsgID:          "ID01",
				StructuredData: "{\"exampleSDID@32473\":{\"eventSource\":\"Application\",\"iut\":\"3\"}}",
				Message:        "Message with structured data",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := rfc5424.NewParser(rfc5424.WithBestEffort())
			syslogMsg, err := parser.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("Failed to parse message: %v", err)
			}

			rfc5424Msg, ok := syslogMsg.(*rfc5424.SyslogMessage)
			if !ok {
				t.Fatal("Parsed message is not a valid RFC5424 message")
			}

			entry := SyslogMessageToLogEntry(rfc5424Msg)
			if entry == nil {
				t.Fatal("SyslogMessageToLogEntry returned nil")
			}

			if entry.Severity != tt.expected.Severity {
				t.Errorf("Severity: got %d, want %d", entry.Severity, tt.expected.Severity)
			}
			if entry.Facility != tt.expected.Facility {
				t.Errorf("Facility: got %d, want %d", entry.Facility, tt.expected.Facility)
			}
			if entry.Version != tt.expected.Version {
				t.Errorf("Version: got %d, want %d", entry.Version, tt.expected.Version)
			}
			if entry.Hostname != tt.expected.Hostname {
				t.Errorf("Hostname: got %q, want %q", entry.Hostname, tt.expected.Hostname)
			}
			if entry.AppName != tt.expected.AppName {
				t.Errorf("AppName: got %q, want %q", entry.AppName, tt.expected.AppName)
			}
			if entry.ProcID != tt.expected.ProcID {
				t.Errorf("ProcID: got %q, want %q", entry.ProcID, tt.expected.ProcID)
			}
			if entry.MsgID != tt.expected.MsgID {
				t.Errorf("MsgID: got %q, want %q", entry.MsgID, tt.expected.MsgID)
			}
			if entry.StructuredData != tt.expected.StructuredData {
				t.Errorf("StructuredData: got %q, want %q", entry.StructuredData, tt.expected.StructuredData)
			}
			if entry.Message != tt.expected.Message {
				t.Errorf("Message: got %q, want %q", entry.Message, tt.expected.Message)
			}

			if entry.Timestamp.IsZero() {
				t.Error("Timestamp should not be zero")
			}
		})
	}
}

func TestSyslogMessageToLogEntryNilHandling(t *testing.T) {
	// Test nil input
	entry := SyslogMessageToLogEntry(nil)
	if entry != nil {
		t.Error("Expected nil entry for nil input")
	}
}

func TestGetFacilityFromPriority(t *testing.T) {
	tests := []struct {
		priority *uint8
		expected uint8
	}{
		{priority: nil, expected: 0},
		{priority: uint8Ptr(0), expected: 0},    // 0 / 8 = 0
		{priority: uint8Ptr(13), expected: 1},   // 13 / 8 = 1
		{priority: uint8Ptr(165), expected: 20}, // 165 / 8 = 20
	}

	for _, tt := range tests {
		result := GetFacilityFromPriority(tt.priority)
		if result != tt.expected {
			priority := "nil"
			if tt.priority != nil {
				priority = string(rune(*tt.priority))
			}
			t.Errorf("GetFacilityFromPriority(%s): got %d, want %d", priority, result, tt.expected)
		}
	}
}

func TestGetSeverityFromPriority(t *testing.T) {
	tests := []struct {
		priority *uint8
		expected uint8
	}{
		{priority: nil, expected: 0},
		{priority: uint8Ptr(0), expected: 0},   // 0 % 8 = 0
		{priority: uint8Ptr(13), expected: 5},  // 13 % 8 = 5
		{priority: uint8Ptr(165), expected: 5}, // 165 % 8 = 5
	}

	for _, tt := range tests {
		result := GetSeverityFromPriority(tt.priority)
		if result != tt.expected {
			priority := "nil"
			if tt.priority != nil {
				priority = string(rune(*tt.priority))
			}
			t.Errorf("GetSeverityFromPriority(%s): got %d, want %d", priority, result, tt.expected)
		}
	}
}

// Helper function to create uint8 pointer
func uint8Ptr(v uint8) *uint8 {
	return &v
}
