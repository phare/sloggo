package formats

import (
	"testing"
	"time"
)

func TestParseRFC3164ToLogEntry_Basic(t *testing.T) {
	line := "<34>Oct 11 22:14:15 mymachine su: 'su root' failed for lonvick on /dev/pts/8"
	entry, err := ParseRFC3164ToLogEntry(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.Facility != 4 || entry.Severity != 2 { // 34 / 8 = 4, 34 % 8 = 2
		t.Errorf("facility/severity mismatch: got (%d,%d)", entry.Facility, entry.Severity)
	}
	if entry.Hostname != "mymachine" {
		t.Errorf("hostname: got %q", entry.Hostname)
	}
	if entry.AppName != "su" {
		t.Errorf("appname: got %q", entry.AppName)
	}
	if entry.ProcID != "-" {
		t.Errorf("procid: got %q", entry.ProcID)
	}
	if entry.Message != "'su root' failed for lonvick on /dev/pts/8" {
		t.Errorf("message: got %q", entry.Message)
	}
	if entry.Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}
}

func TestParseRFC3164ToLogEntry_WithPID(t *testing.T) {
	line := "<190>Nov  6 09:01:02 esphome-device esphome[1234]: Sensor reading: 42"
	entry, err := ParseRFC3164ToLogEntry(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.Facility != 23 || entry.Severity != 6 { // 190 / 8 = 23, 190 % 8 = 6
		t.Errorf("facility/severity mismatch: got (%d,%d)", entry.Facility, entry.Severity)
	}
	if entry.Hostname != "esphome-device" {
		t.Errorf("hostname: got %q", entry.Hostname)
	}
	if entry.AppName != "esphome" {
		t.Errorf("appname: got %q", entry.AppName)
	}
	if entry.ProcID != "1234" {
		t.Errorf("procid: got %q", entry.ProcID)
	}
	if entry.Message != "Sensor reading: 42" {
		t.Errorf("message: got %q", entry.Message)
	}
}

func TestParseRFC3164ToLogEntry_MultilineMessage(t *testing.T) {
	line := "<134>Feb  1 11:37:00 modbus-ble-bridge mdns: [C][mdns:124]: mDNS:\n\n  Hostname: modbus-ble-bridge"
	entry, err := ParseRFC3164ToLogEntry(line)
	if err != nil {
		t.Fatalf("unexpected error parsing multiline: %v", err)
	}
	if entry.Facility != 16 || entry.Severity != 6 { // 134 / 8 = 16, 134 % 8 = 6
		t.Errorf("facility/severity mismatch: got (%d,%d)", entry.Facility, entry.Severity)
	}
	if entry.Hostname != "modbus-ble-bridge" {
		t.Errorf("hostname: got %q", entry.Hostname)
	}
	if entry.AppName != "mdns" {
		t.Errorf("appname: got %q", entry.AppName)
	}
	expectedMsg := "[C][mdns:124]: mDNS:\n\n  Hostname: modbus-ble-bridge"
	if entry.Message != expectedMsg {
		t.Errorf("message mismatch:\nexpected: %q\n     got: %q", expectedMsg, entry.Message)
	}
}

func TestParseRFC3164ToLogEntry_YearBoundary(t *testing.T) {
	// Test year boundary handling: December logs received in January
	now := time.Now()

	line := "<34>Dec 31 23:59:59 testhost app: Year boundary test"
	entry, err := ParseRFC3164ToLogEntry(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// If we're in January and the log says December, it must be from last year
	if now.Month() == time.January {
		if entry.Timestamp.Month() != time.December {
			t.Errorf("expected December month, got %v", entry.Timestamp.Month())
		}
		expectedYear := now.Year() - 1
		if entry.Timestamp.Year() != expectedYear {
			t.Errorf("expected year %d for December log in January, got %d", expectedYear, entry.Timestamp.Year())
		}
	}

	// Verify the parsed values
	if entry.Hostname != "testhost" {
		t.Errorf("hostname: got %q", entry.Hostname)
	}
	if entry.AppName != "app" {
		t.Errorf("appname: got %q", entry.AppName)
	}
}

func TestParseRFC3164ToLogEntry_InvalidPriority(t *testing.T) {
	// Test priority out of range
	testCases := []struct {
		name string
		line string
	}{
		{"priority too high", "<192>Oct 11 22:14:15 mymachine su: test"},
		{"priority too high 2", "<999>Oct 11 22:14:15 mymachine su: test"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseRFC3164ToLogEntry(tc.line)
			if err == nil {
				t.Error("expected error for invalid priority, got nil")
			}
		})
	}
}

func TestParseRFC3164ToLogEntry_InvalidTimestamp(t *testing.T) {
	// Test invalid timestamp format
	line := "<34>Invalid 99 99:99:99 mymachine su: test"
	_, err := ParseRFC3164ToLogEntry(line)
	if err == nil {
		t.Error("expected error for invalid timestamp, got nil")
	}
}
