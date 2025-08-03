package utils

import (
	"os"
	"strings"
)

// The following variables are set at build time (see GitHub Action & Makefile)

var Listeners []string

var UdpPort string

var TcpPort string

var Version string

func init() {
	Listeners = strings.Split(GetSanitizedEnv("SLOGGO_LISTENERS", "tcp,udp"), ",")
	UdpPort = GetSanitizedEnv("SLOGGO_UDP_PORT", "5514")
	TcpPort = GetSanitizedEnv("SLOGGO_TCP_PORT", "6514")
}

func GetSanitizedEnv(key string, defaultValue string) string {
	value := os.Getenv(key)

	if value == "" {
		return defaultValue
	}

	value = strings.TrimSpace(value)
	value = strings.ToLower(value)

	return value
}
