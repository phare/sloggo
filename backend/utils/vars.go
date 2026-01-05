package utils

import (
	"os"
	"strconv"
	"strings"
	"sync"
)

// The following variables are set at build time (see GitHub Action & Makefile)

var Listeners []string

var UdpPort string

var TcpPort string

var ApiPort string

var LogRetentionMinutes int64

var Debug bool

var Version string // Set via -X flag during build

// logFormat controls how incoming syslog messages are parsed.
// Supported values (case-insensitive):
//   - "auto"   : try RFC5424 first, then RFC3164 (default)
//   - "rfc5424": only parse as RFC5424
//   - "rfc3164": only parse as RFC3164
// Any other value falls back to "auto".
var logFormat string
var logFormatMutex sync.RWMutex

// GetLogFormat returns the current log format in a thread-safe manner
func GetLogFormat() string {
	logFormatMutex.RLock()
	defer logFormatMutex.RUnlock()
	return logFormat
}

// SetLogFormat sets the log format in a thread-safe manner
func SetLogFormat(format string) {
	logFormatMutex.Lock()
	defer logFormatMutex.Unlock()
	logFormat = format
}

func init() {
	Listeners = strings.Split(GetSanitizedEnvString("SLOGGO_LISTENERS", "tcp,udp"), ",")
	UdpPort = GetSanitizedEnvString("SLOGGO_UDP_PORT", "5514")
	TcpPort = GetSanitizedEnvString("SLOGGO_TCP_PORT", "6514")
	ApiPort = GetSanitizedEnvString("SLOGGO_API_PORT", "8080")
	LogRetentionMinutes = GetSanitizedEnvInt64("SLOGGO_LOG_RETENTION_MINUTES", 30*24*60) // Default to 30 days
	Debug = GetSanitizedEnvString("SLOGGO_DEBUG", "false") == "true"

	// Configure log format selection
	switch GetSanitizedEnvString("SLOGGO_LOG_FORMAT", "auto") {
	case "rfc5424":
		logFormat = "rfc5424"
	case "rfc3164":
		logFormat = "rfc3164"
	default:
		logFormat = "auto"
	}
}

func GetSanitizedEnvString(key string, defaultValue string) string {
	value := os.Getenv(key)

	if value == "" {
		return defaultValue
	}

	value = strings.TrimSpace(value)
	value = strings.ToLower(value)

	return value
}

func GetSanitizedEnvInt64(key string, defaultValue int64) int64 {
	value := os.Getenv(key)

	if value == "" {
		return defaultValue
	}

	value = strings.TrimSpace(value)

	// Convert string to int64
	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultValue
	}

	return intValue
}
