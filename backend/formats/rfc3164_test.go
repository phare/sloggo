package formats

import (
	"testing"
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
