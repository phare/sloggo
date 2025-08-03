package listener

import (
	"log"
	"net"
	"sloggo/db"
	"sloggo/formats"
	"sloggo/utils"
	"time"
)

func StartTCPListener() {
	port := utils.TcpPort

	_, err := net.LookupPort("tcp", port)
	if err != nil {
		log.Fatalf("Invalid TCP port %s: %v", port, err)
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to start TCP listener on port %s: %v", port, err)
	}
	defer listener.Close()

	log.Printf("TCP listener is running on port :%s", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting TCP connection: %v", err)
			continue
		}
		go handleTCPConnection(conn)
	}
}

func handleTCPConnection(conn net.Conn) {
	defer conn.Close()

	// Set read deadline to prevent hanging
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		log.Printf("Error setting read deadline: %v", err)
		return
	}

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			log.Printf("Connection read timed out")
		} else {
			log.Printf("Error reading from TCP connection: %v", err)
		}
		return
	}

	// Process and store the message
	message := string(buffer[:n])
	logMessage, err := formats.NewRFC5424Log(message)
	if err != nil {
		log.Printf("Failed to process log message: %v", err)
		return
	}

	query, params := logMessage.ToSQL()

	db.StoreLog(query, params)
}
