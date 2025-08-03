# Sloggo Benchmarking Tools

This directory contains a benchmarking script for Sloggo, used to test log ingestion under a high burst scenario.

#### Build

```sh
go build -o sloggo-bench
```

#### Usage

```sh
./sloggo-bench --host=localhost --port=6514 --protocol=tcp --total=1000000 --workers=16 --batch-size=1000
```

#### Options

- `--host`: Target host (default: "127.0.0.1")
- `--port`: Target port (default: 6514)
- `--protocol`: Protocol, tcp or udp (default: "tcp")
- `--total`: Total number of logs to send (default: 100000)
- `--workers`: Number of worker goroutines (default: number of CPU cores)
- `--batch-size`: Number of logs per batch (default: 1000)
- `--app`: Application name for syslog (default: "sloggo-bench")
- `--hostname`: Hostname for syslog (default: system hostname)
- `--facility`: Syslog facility code (default: 1)
- `--severity`: Syslog severity code (default: 6)
