package main

import (
	"slices"
	"sloggo/server"
	"sloggo/utils"

	"sloggo/listener"
)

func main() {
	if slices.Contains(utils.Listeners, "udp") {
		go listener.StartUDPListener()
	}

	if slices.Contains(utils.Listeners, "tcp") {
		go listener.StartTCPListener()
	}

	server.StartHTTPServer()
}
