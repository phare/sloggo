package models

import (
	"time"
)

// LogEntry represents a log entry in the system
// It's used both for database operations and API responses
type LogEntry struct {
	// Core fields mapped directly from database
	RowID          int64     `json:"id"` // Built-in unique identifier
	Facility       uint8     `json:"facility"`
	Severity       uint8     `json:"severity"`
	Version        uint16    `json:"version,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
	Hostname       string    `json:"hostname"`
	AppName        string    `json:"appName"` // Note: DB column is app_name
	ProcID         string    `json:"procId"`  // Note: DB column is procid
	MsgID          string    `json:"msgId"`   // Note: DB column is msgid
	StructuredData string    `json:"-"`       // Note: DB column is structured_data
	Message        string    `json:"message"` // Note: DB column is msg

	// Derived fields for API responses
	ParsedStructuredData map[string]map[string]string `json:"structuredData,omitempty"` // Parsed form of StructuredData
}
