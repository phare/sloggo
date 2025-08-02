package formats

// LogEntry represents a structured log message that can be stored in the database.
type LogEntry interface {
	// Severity returns the log message severity level.
	Severity() int

	// Facility returns the log message facility code.
	Facility() int

	// String returns a string representation of the log message.
	String() string
}

// LogParser defines an interface for parsing different log formats.
type LogParser interface {
	// Parse attempts to parse a log message string into a LogEntry.
	// Returns an error if the message cannot be parsed.
	Parse(message string) (LogEntry, error)

	// Name returns the name of the log format this parser handles.
	Name() string
}

// LogStore defines the interface for storing log entries.
type LogStore interface {
	// Store persists a log entry to the database.
	Store(entry LogEntry) error

	// Query retrieves log entries based on query parameters.
	Query(params QueryParams) ([]LogEntry, error)
}

// QueryParams defines the parameters for querying log entries.
type QueryParams struct {
	Facility      *int   // Filter by facility code
	Severity      *int   // Filter by severity level
	FromTimestamp string // Filter by timestamp range start
	ToTimestamp   string // Filter by timestamp range end
	Hostname      string // Filter by hostname
	AppName       string // Filter by application name
	Limit         int    // Maximum number of entries to return
	Offset        int    // Number of entries to skip
	OrderBy       string // Field to order by
	OrderDesc     bool   // If true, order in descending order
}

// DefaultQueryParams returns a QueryParams with default values.
func DefaultQueryParams() QueryParams {
	return QueryParams{
		Limit:     100,
		OrderBy:   "timestamp",
		OrderDesc: true,
	}
}

// RegisteredParsers maintains a map of available log parsers.
var RegisteredParsers = make(map[string]LogParser)

// RegisterParser adds a new log parser to the registry.
func RegisterParser(parser LogParser) {
	RegisteredParsers[parser.Name()] = parser
}

// GetParser returns a parser by name, or nil if not found.
func GetParser(name string) LogParser {
	return RegisteredParsers[name]
}
