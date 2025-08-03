package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Configuration options
var (
	host      string
	port      int
	protocol  string
	total     int
	workers   int
	batchSize int
	appName   string
	hostname  string
	facility  int
	severity  int
)

// Statistics
var (
	startTime  time.Time
	sentLogs   int64
	errorCount int64
)

func init() {
	// Get hostname or use default
	defaultHostname, err := os.Hostname()
	if err != nil {
		defaultHostname = "sloggo-bench"
	}

	// Parse command line flags
	flag.StringVar(&host, "host", "127.0.0.1", "Target host")
	flag.IntVar(&port, "port", 6514, "Target port")
	flag.StringVar(&protocol, "protocol", "tcp", "Protocol (tcp or udp)")
	flag.IntVar(&total, "total", 100000, "Total number of logs to send")
	flag.IntVar(&workers, "workers", runtime.NumCPU(), "Number of worker goroutines")
	flag.IntVar(&batchSize, "batch-size", 1000, "Number of logs per batch")
	flag.StringVar(&appName, "app", "sloggo-bench", "Application name for syslog")
	flag.StringVar(&hostname, "hostname", defaultHostname, "Hostname for syslog")
	flag.IntVar(&facility, "facility", 1, "Syslog facility code")
	flag.IntVar(&severity, "severity", 6, "Syslog severity code")
	flag.Parse()

	// Validate parameters
	if workers < 1 {
		workers = 1
	}
	if batchSize < 1 {
		batchSize = 1
	}
	if total < 1 {
		total = 1
	}
}

func main() {
	// Display banner
	fmt.Println("=================================================================")
	fmt.Println("🚀 Sloggo burst ingestion benchmark")
	fmt.Println("=================================================================")
	fmt.Printf("Target:      %s:%d (%s)\n", host, port, protocol)
	fmt.Printf("Logs:        %d\n", total)
	fmt.Printf("Workers:     %d\n", workers)
	fmt.Printf("Batch size:  %d\n", batchSize)
	fmt.Printf("Syslog:      facility=%d, severity=%d, app=%s\n", facility, severity, appName)
	fmt.Println("=================================================================")

	// Create a wait group to track worker completion
	var wg sync.WaitGroup

	// Calculate the number of logs per worker
	logsPerWorker := total / workers
	remainder := total % workers

	// Record start time
	startTime = time.Now()

	// Start workers
	for i := 0; i < workers; i++ {
		// Calculate the range for this worker
		workerLogs := logsPerWorker
		if i < remainder {
			workerLogs++
		}

		wg.Add(1)
		go func(workerID, numLogs int) {
			defer wg.Done()
			if protocol == "tcp" {
				sendTCPLogs(workerID, numLogs)
			} else {
				sendUDPLogs(workerID, numLogs)
			}
		}(i, workerLogs)
	}

	// Wait for all workers to complete
	wg.Wait()
	fmt.Println() // Newline after progress updates

	// Calculate and display results
	duration := time.Since(startTime)
	logsPerSecond := float64(sentLogs) / duration.Seconds()

	fmt.Println("=================================================================")
	fmt.Printf("✅ Benchmark complete!\n")
	fmt.Printf("Duration:    %.2f seconds\n", duration.Seconds())
	fmt.Printf("Sent logs:   %d\n", sentLogs)
	fmt.Printf("Errors:      %d\n", errorCount)
	fmt.Printf("Throughput:  %.2f logs/second\n", logsPerSecond)
	fmt.Println("=================================================================")
}

// sendTCPLogs sends logs using TCP protocol
func sendTCPLogs(workerID, numLogs int) {
	// Calculate priority
	priority := facility*8 + severity

	// Target address
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	// Process batches of logs
	remaining := numLogs
	for remaining > 0 {
		// Determine batch size for this iteration
		currentBatch := min(remaining, batchSize)

		// Connect to the server
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			atomic.AddInt64(&errorCount, 1)
			log.Printf("Worker %d: TCP connection error: %v\n", workerID, err)
			time.Sleep(100 * time.Millisecond) // Brief pause before retry
			continue
		}

		// Set a deadline for the connection
		conn.SetDeadline(time.Now().Add(10 * time.Second))

		// Build batch of messages
		var builder strings.Builder
		for i := range currentBatch {
			timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
			msgID := fmt.Sprintf("MSG%d-%d", workerID, i)
			logline := fmt.Sprintf("<%d>1 %s %s %s %d %s - Log message %d from worker %d\n",
				priority, timestamp, hostname, appName, os.Getpid(), msgID, i, workerID)
			builder.WriteString(logline)
		}

		// Send the batch
		_, err = conn.Write([]byte(builder.String()))
		conn.Close()

		if err != nil {
			atomic.AddInt64(&errorCount, int64(currentBatch))
			log.Printf("Worker %d: TCP send error: %v\n", workerID, err)
		} else {
			atomic.AddInt64(&sentLogs, int64(currentBatch))
			remaining -= currentBatch
		}
	}
}

// sendUDPLogs sends logs using UDP protocol
func sendUDPLogs(workerID, numLogs int) {
	// Calculate priority
	priority := facility*8 + severity

	// Resolve UDP address
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		atomic.AddInt64(&errorCount, int64(numLogs))
		log.Printf("Worker %d: UDP address resolution error: %v\n", workerID, err)
		return
	}

	// Create UDP connection
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		atomic.AddInt64(&errorCount, int64(numLogs))
		log.Printf("Worker %d: UDP connection error: %v\n", workerID, err)
		return
	}
	defer conn.Close()

	// Process batches of logs
	remaining := numLogs
	for remaining > 0 {
		// Determine batch size for this iteration
		currentBatch := min(remaining, batchSize)

		// Send individual UDP messages
		for i := range currentBatch {
			timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
			msgID := fmt.Sprintf("MSG%d-%d", workerID, i)
			logline := fmt.Sprintf("<%d>1 %s %s %s %d %s - Log message %d from worker %d\n",
				priority, timestamp, hostname, appName, os.Getpid(), msgID, i, workerID)

			_, err := conn.Write([]byte(logline))
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
			} else {
				atomic.AddInt64(&sentLogs, 1)
			}
		}

		remaining -= currentBatch
	}
}
