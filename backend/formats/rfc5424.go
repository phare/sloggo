package formats

import (
	"encoding/json"
	"log"
	"sloggo/models"
	"time"

	"github.com/leodido/go-syslog/v4/rfc5424"
)

// GetFacilityFromPriority extracts the facility from a syslog priority value
func GetFacilityFromPriority(priority *uint8) uint8 {
	if priority == nil {
		return 0
	}
	return *priority / 8
}

// GetSeverityFromPriority extracts the severity from a syslog priority value
func GetSeverityFromPriority(priority *uint8) uint8 {
	if priority == nil {
		return 0
	}
	return *priority % 8
}

// SyslogMessageToLogEntry converts a SyslogMessage to LogEntry struct for efficient DuckDB insertion
func SyslogMessageToLogEntry(msg *rfc5424.SyslogMessage) *models.LogEntry {
	if msg == nil {
		return nil
	}

	// Calculate facility and severity from priority
	var facility, severity uint8
	if msg.Priority != nil {
		facility = GetFacilityFromPriority(msg.Priority)
		severity = GetSeverityFromPriority(msg.Priority)
	}

	// Use timestamp from message or current time
	var timestamp time.Time
	if msg.Timestamp != nil {
		timestamp = *msg.Timestamp
	} else {
		// Use current time if timestamp is missing
		timestamp = time.Now()
	}

	// Use default values for nil pointers
	hostname := "-"
	if msg.Hostname != nil {
		hostname = *msg.Hostname
	}

	appName := "-"
	if msg.Appname != nil {
		appName = *msg.Appname
	}

	procId := "-"
	if msg.ProcID != nil {
		procId = *msg.ProcID
	}

	msgId := "-"
	if msg.MsgID != nil {
		msgId = *msg.MsgID
	}

	// Format structured data
	structuredData := "-"
	if msg.StructuredData != nil && len(*msg.StructuredData) > 0 {
		structuredData = formatStructuredData(*msg.StructuredData)
	}

	// Get message content
	msgContent := ""
	if msg.Message != nil {
		msgContent = *msg.Message
	}

	// Create the entry
	entry := &models.LogEntry{
		Severity:       severity,
		Facility:       facility,
		Version:        msg.Version,
		Timestamp:      timestamp,
		Hostname:       hostname,
		AppName:        appName,
		ProcID:         procId,
		MsgID:          msgId,
		StructuredData: structuredData,
		Message:        msgContent,
	}

	return entry
}

// formatStructuredData converts the structured data map to a json string format
func formatStructuredData(structData map[string]map[string]string) string {
	jsonBytes, err := json.Marshal(structData)
	if err != nil {
		log.Printf("Failed to marshal structured data: %v", err)
		return "{}"
	}

	return string(jsonBytes)
}
