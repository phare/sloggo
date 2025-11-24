package main

import (
	"sloggo/server"
	"sloggo/utils"

	"sloggo/listener"
)

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func main() {
	if contains(utils.Listeners, "udp") {
		go listener.StartUDPListener()
	}

	if contains(utils.Listeners, "tcp") {
		go listener.StartTCPListener()
	}

	server.StartHTTPServer()
}
