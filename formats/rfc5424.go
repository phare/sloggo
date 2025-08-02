package formats

import (
	"fmt"
	"strings"
)

// RFC5424Log represents a structured log message in RFC 5424 format.
type RFC5424Log struct {
	facility       int
	severity       int
	version        int
	timestamp      string
	hostname       string
	appName        string
	procId         string
	msgId          string
	structuredData string
	msg            string
}

// NewRFC5424Log creates a new LogEntry from an RFC 5424 log message.
func NewRFC5424Log(message string) (*RFC5424Log, error) {
	// Remove trailing newline if present
	message = strings.TrimSuffix(message, "\n")

	// Extract and validate PRI
	if !strings.HasPrefix(message, "<") {
		return nil, fmt.Errorf("message must start with PRI")
	}
	end := strings.Index(message, ">")
	if end == -1 {
		return nil, fmt.Errorf("invalid PRI format: missing closing >")
	}

	var pri int
	_, err := fmt.Sscanf(message[1:end], "%d", &pri)
	if err != nil {
		return nil, fmt.Errorf("invalid PRI value: %v", err)
	}
	message = message[end+1:]

	// Extract version
	parts := strings.SplitN(strings.TrimSpace(message), " ", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("missing version or other fields")
	}

	var version int
	_, err = fmt.Sscanf(parts[0], "%d", &version)
	if err != nil {
		return nil, fmt.Errorf("invalid version value: %v", err)
	}
	message = parts[1]

	// Split header fields
	parts = strings.SplitN(strings.TrimSpace(message), " ", 6)
	if len(parts) < 6 {
		return nil, fmt.Errorf("incomplete message header")
	}

	timestamp := parts[0]
	hostname := parts[1]
	appName := parts[2]
	procId := parts[3]
	msgId := parts[4]
	remainder := parts[5]

	// Handle structured data and message
	var structuredData, msg string
	if strings.HasPrefix(remainder, "-") {
		structuredData = "-"
		msg = strings.TrimPrefix(remainder, "- ")
	} else {
		endSD := strings.Index(remainder, "] ")
		if endSD == -1 {
			return nil, fmt.Errorf("malformed structured data")
		}
		structuredData = remainder[:endSD+1]
		msg = strings.TrimSpace(remainder[endSD+2:])
	}

	facility := pri / 8
	severity := pri % 8

	return &RFC5424Log{
		facility:       facility,
		severity:       severity,
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
	return r.facility
}

// Severity returns the log message severity level.
func (r *RFC5424Log) Severity() int {
	return r.severity
}

// Version returns the protocol version.
func (r *RFC5424Log) Version() int {
	return r.version
}

// Timestamp returns the message timestamp.
func (r *RFC5424Log) Timestamp() string {
	return r.timestamp
}

// Hostname returns the message hostname.
func (r *RFC5424Log) Hostname() string {
	return r.hostname
}

// AppName returns the application name.
func (r *RFC5424Log) AppName() string {
	return r.appName
}

// ProcID returns the process ID.
func (r *RFC5424Log) ProcID() string {
	return r.procId
}

// MsgID returns the message ID.
func (r *RFC5424Log) MsgID() string {
	return r.msgId
}

// StructuredData returns the structured data.
func (r *RFC5424Log) StructuredData() string {
	return r.structuredData
}

// Message returns the log message.
func (r *RFC5424Log) Message() string {
	return r.msg
}

// ToSQLInsert returns a SQL insert statement and parameters for this log entry.
func (r *RFC5424Log) ToSQL() (string, []any) {
	query := `
		INSERT INTO logs (
			facility, severity, version, timestamp,
			hostname, app_name, procid, msgid,
			structured_data, msg
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	params := []any{
		r.facility,
		r.severity,
		r.version,
		r.timestamp,
		r.hostname,
		r.appName,
		r.procId,
		r.msgId,
		r.structuredData,
		r.msg,
	}

	return query, params
}
