package main

import (
	"log"
	"slices"
	"sloggo/server"
	"sloggo/utils"

	"sloggo/listener"
)

func main() {
	// Startup configuration log
	log.Printf("Sloggo version: %s", utils.Version)
	log.Printf("Config: listeners=%v udp_port=%s tcp_port=%s api_port=%s", utils.Listeners, utils.UdpPort, utils.TcpPort, utils.ApiPort)
	log.Printf("Config: log_format=%s debug=%t retention_minutes=%d", utils.GetLogFormat(), utils.Debug, utils.LogRetentionMinutes)

	if slices.Contains(utils.Listeners, "udp") {
		go listener.StartUDPListener()
	}

	if slices.Contains(utils.Listeners, "tcp") {
		go listener.StartTCPListener()
	}

	server.StartHTTPServer()
}
