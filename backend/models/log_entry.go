package models

import (
	"time"
)

// LogEntry represents a log entry in the system
// It's used both for database operations and API responses
type LogEntry struct {
	// Core fields mapped directly from database
	RowID          int64     `json:"id"` // SQLite's built-in unique identifier
	Facility       int       `json:"facility"`
	Severity       int       `json:"severity"`
	Timestamp      time.Time `json:"timestamp"`
	Hostname       string    `json:"hostname"`
	AppName        string    `json:"appName"` // Note: DB column is app_name
	ProcID         string    `json:"procId"`  // Note: DB column is procid
	MsgID          string    `json:"msgId"`   // Note: DB column is msgid
	StructuredData string    `json:"-"`       // Raw structured data from DB (structured_data column)
	Message        string    `json:"message"` // Note: DB column is msg

	// Derived fields for API responses
	ParsedStructuredData map[string]map[string]string `json:"structuredData,omitempty"` // Parsed form of StructuredData
}
