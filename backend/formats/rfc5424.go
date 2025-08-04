package formats

import (
	"encoding/json"
	"fmt"
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

// SyslogMessageToSQL converts a SyslogMessage to a SQL statement and parameters
func SyslogMessageToSQL(msg *rfc5424.SyslogMessage) (string, []any) {
	query := `
		INSERT INTO logs (
			facility, severity, version, timestamp,
			hostname, app_name, procid, msgid,
			structured_data, msg
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	// Calculate facility and severity from priority
	var facility, severity uint8
	if msg.Priority != nil {
		facility = GetFacilityFromPriority(msg.Priority)
		severity = GetSeverityFromPriority(msg.Priority)
	}

	// Convert timestamp to string if it exists
	var timestampStr string
	if msg.Timestamp != nil {
		timestampStr = msg.Timestamp.Format(time.RFC3339Nano)
	} else {
		// Use current time if timestamp is missing
		timestampStr = time.Now().Format(time.RFC3339Nano)
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

	// Create parameters for SQL query
	params := []any{
		facility,
		severity,
		msg.Version,
		timestampStr,
		hostname,
		appName,
		procId,
		msgId,
		structuredData,
		msgContent,
	}

	return query, params
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

// ParseRFC5424Message parses an RFC5424 syslog message string
func ParseRFC5424Message(message string) (*rfc5424.SyslogMessage, error) {
	// Create a new parser with best effort mode
	parser := rfc5424.NewParser(rfc5424.WithBestEffort())

	// Parse the message
	syslogMsg, err := parser.Parse([]byte(message))
	if err != nil {
		return nil, fmt.Errorf("failed to parse RFC5424 message: %v", err)
	}

	// Convert to RFC5424 message
	rfc5424Msg, ok := syslogMsg.(*rfc5424.SyslogMessage)
	if !ok {
		return nil, fmt.Errorf("parsed message is not a valid RFC5424 message")
	}

	return rfc5424Msg, nil
}
