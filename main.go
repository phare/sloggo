package main

import (
	"sloggo/server"

	"sloggo/listener"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Start UDP listener
	go listener.StartUDPListener()

	// Start TCP listener
	go listener.StartTCPListener()

	// Start HTTP server
	server.StartHTTPServer()
}
