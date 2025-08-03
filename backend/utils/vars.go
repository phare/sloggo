package utils

import (
	"os"
	"strconv"
	"strings"
)

// The following variables are set at build time (see GitHub Action & Makefile)

var Listeners []string

var UdpPort string

var TcpPort string

var LogRetentionMinutes int64

var Version string

func init() {
	Listeners = strings.Split(GetSanitizedEnvString("SLOGGO_LISTENERS", "tcp,udp"), ",")
	UdpPort = GetSanitizedEnvString("SLOGGO_UDP_PORT", "5514")
	TcpPort = GetSanitizedEnvString("SLOGGO_TCP_PORT", "6514")
	LogRetentionMinutes = GetSanitizedEnvInt64("SLOGGO_LOG_RETENTION_MINUTES", 3*24*60) // Default to 3 days
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
