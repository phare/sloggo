package formats

import (
	"fmt"
	"strings"
	"time"

	"github.com/leodido/go-syslog/v4/rfc5424"
)

// RFC5424Log represents a structured log message in RFC 5424 format.
type RFC5424Log struct {
	facility       *uint8
	severity       *uint8
	priority       *uint8
	version        int
	timestamp      *time.Time
	hostname       *string
	appName        *string
	procId         *string
	msgId          *string
	structuredData string
	msg            *string
}

// NewRFC5424Log creates a new LogEntry from an RFC 5424 log message.
func NewRFC5424Log(message string) (*RFC5424Log, error) {
	// Remove trailing newline if present
	message = strings.TrimSuffix(message, "\n")

	// Create a new parser with best effort mode
	parser := rfc5424.NewParser(rfc5424.WithBestEffort())

	// Parse the message
	parsedMessage, err := parser.Parse([]byte(message))
	if err != nil {
		return nil, fmt.Errorf("failed to parse RFC5424 message: %v", err)
	}

	// Convert to RFC5424 message
	rfc5424Message, ok := parsedMessage.(*rfc5424.SyslogMessage)
	if !ok {
		return nil, fmt.Errorf("parsed message is not a valid RFC5424 message")
	}

	// Extract facility and severity from priority
	priority := rfc5424Message.Priority
	if priority == nil {
		return nil, fmt.Errorf("missing priority in log message")
	}

	var facility, severity uint8
	facility = *priority / 8
	severity = *priority % 8

	facilityPtr := &facility
	severityPtr := &severity

	// Get timestamp
	var timestamp *time.Time
	if rfc5424Message.Timestamp != nil {
		timestamp = rfc5424Message.Timestamp
	} else {
		now := time.Now()
		timestamp = &now
	}

	// Extract hostname
	var hostname *string
	if rfc5424Message.Hostname != nil {
		hostname = rfc5424Message.Hostname
	}

	// Extract app name
	var appName *string
	if rfc5424Message.Appname != nil {
		appName = rfc5424Message.Appname
	}

	// Extract process ID
	var procId *string
	if rfc5424Message.ProcID != nil {
		procId = rfc5424Message.ProcID
	}

	// Extract message ID
	var msgId *string
	if rfc5424Message.MsgID != nil {
		msgId = rfc5424Message.MsgID
	}

	// Extract structured data
	structuredData := "-"
	if rfc5424Message.StructuredData != nil && len(*rfc5424Message.StructuredData) > 0 {
		// Convert structured data to string format
		parts := make([]string, 0, len(*rfc5424Message.StructuredData))
		for id, params := range *rfc5424Message.StructuredData {
			paramStrings := make([]string, 0, len(params))
			for name, value := range params {
				paramStrings = append(paramStrings, fmt.Sprintf(`%s="%s"`, name, value))
			}

			parts = append(parts, fmt.Sprintf("[%s %s]", id, strings.Join(paramStrings, " ")))
		}

		if len(parts) > 0 {
			structuredData = strings.Join(parts, "")
		}
	}

	// Extract message
	var msg *string
	if rfc5424Message.Message != nil {
		msg = rfc5424Message.Message
	}

	// Create and return the RFC5424Log struct
	version := 1
	if rfc5424Message.Version != 0 {
		version = int(rfc5424Message.Version)
	}

	return &RFC5424Log{
		facility:       facilityPtr,
		severity:       severityPtr,
		priority:       priority,
		version:        version,
		timestamp:      timestamp,
		hostname:       hostname,
		appName:        appName,
		procId:         procId,
		msgId:          msgId,
		structuredData: structuredData,
		msg:            msg,
	}, nil
}

// Facility returns the log message facility code.
func (r *RFC5424Log) Facility() int {
	if r.facility == nil {
		return 0
	}
	return int(*r.facility)
}

// Severity returns the log message severity level.
func (r *RFC5424Log) Severity() int {
	if r.severity == nil {
		return 0
	}
	return int(*r.severity)
}

// Version returns the protocol version.
func (r *RFC5424Log) Version() int {
	return r.version
}

// Timestamp returns the message timestamp.
func (r *RFC5424Log) Timestamp() *time.Time {
	return r.timestamp
}

// Hostname returns the message hostname.
func (r *RFC5424Log) Hostname() *string {
	return r.hostname
}

// AppName returns the application name.
func (r *RFC5424Log) AppName() *string {
	return r.appName
}

// ProcID returns the process ID.
func (r *RFC5424Log) ProcID() *string {
	return r.procId
}

// MsgID returns the message ID.
func (r *RFC5424Log) MsgID() *string {
	return r.msgId
}

// StructuredData returns the structured data.
func (r *RFC5424Log) StructuredData() string {
	return r.structuredData
}

// Message returns the log message.
func (r *RFC5424Log) Message() *string {
	return r.msg
}

// ToSQL returns a SQL insert statement and parameters for this log entry.
func (r *RFC5424Log) ToSQL() (string, []any) {
	query := `
		INSERT INTO logs (
			facility, severity, version, timestamp,
			hostname, app_name, procid, msgid,
			structured_data, msg
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	// Convert timestamp to string if it exists
	var timestampStr string
	if r.timestamp != nil {
		timestampStr = r.timestamp.Format(time.RFC3339)
	}

	// Use default values for nil pointers
	hostname := "-"
	if r.hostname != nil {
		hostname = *r.hostname
	}

	appName := "-"
	if r.appName != nil {
		appName = *r.appName
	}

	procId := "-"
	if r.procId != nil {
		procId = *r.procId
	}

	msgId := "-"
	if r.msgId != nil {
		msgId = *r.msgId
	}

	msg := ""
	if r.msg != nil {
		msg = *r.msg
	}

	params := []any{
		r.Facility(),
		r.Severity(),
		r.version,
		timestampStr,
		hostname,
		appName,
		procId,
		msgId,
		r.structuredData,
		msg,
	}

	return query, params
}
