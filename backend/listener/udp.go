package listener

import (
	"log"
	"net"
	"os"
	"sloggo/db"
	"sloggo/formats"
)

// StartUDPListener starts a UDP listener on the specified port.
// The port can be configured using the environment variable "UDP_PORT".
// Logs received are stored in the provided SQLite database.
func StartUDPListener() {
	port := os.Getenv("UDP_PORT")

	if port == "" {
		port = "5514"
	}

	intPort, err := net.LookupPort("udp", port)
	if err != nil {
		log.Fatalf("Invalid UDP port %s: %v", port, err)
	}

	addr := net.UDPAddr{
		Port: intPort,
		IP:   net.ParseIP("0.0.0.0"),
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalf("Failed to start UDP listener on port %s: %v", port, err)
	}
	defer conn.Close()

	log.Printf("UDP listener is running on port :%s", port)

	buffer := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error reading from UDP: %v", err)
			continue
		}

		logEntry, err := formats.NewRFC5424Log(string(buffer[:n]))
		if err != nil {
			log.Printf("Failed to parse log message: %v", err)
			continue
		}

		query, params := logEntry.ToSQL()

		if err := db.StoreLog(query, params); err != nil {
			log.Printf("Failed to store log message: %v", err)
			continue
		}
	}
}
