<p align="center"><img src="/sloggo-logo.png" width="300" alt="Sloggo Logo"></p>

<br />

<h1 align="center">Sloggo</h1>
<p align="center">
<a href="https://github.com/phare/sloggo/actions/workflows/build.yml"><img src="https://img.shields.io/github/actions/workflow/status/phare/sloggo/build" alt="Build"></a>
<a href="https://github.com/phare/sloggo/tags"><img src="https://img.shields.io/github/v/tag/phare/sloggo" alt="Version"></a>
<a href="https://github.com/phare/sloggo?tab=MIT-1-ov-file#readme"><img src="https://img.shields.io/github/license/phare/sloggo" alt="License"></a>
</p>

<div align="center">
    Minimal RFC 5424 syslog collector and viewer. SQLite-backed. Runs as a single, resource-friendly process.
</div>

<br />
<br />

# Introduction

Sloggo is a lightweight log collection and exploration tool. It ingests logs over TCP and UDP using the RFC 5424 Syslog protocol, stores them in SQLite, and presents them in a clean, modern web UI.

Designed for small to medium-sized setups where you want real-time logs without spinning up the JVM or a full Kubernetes cluster to ingest 10 daily lines of logs.

It runs in a single process with minimal resource usage and starts instantly via Docker.

> [!WARNING]
> Sloggo is currently in alpha release, do not use it for anything serious, it also doesn’t offer any security layer by default, only use it in a private network, or behind a secure reverse proxy.

## Getting Started

1. Start the container with docker or podman:

```bash
docker run -p 5514:5514/udp -p 6514:6514 -p 8080:8080 \
  -e SLOGGO_LISTENERS=tcp,udp \
  -e SLOGGO_UDP_PORT=5514 \
  -e SLOGGO_TCP_PORT=6514 \
  -e SLOGGO_API_PORT=8080 \
   ghcr.io/phare/sloggo:latest
```

2. Send some logs

```bash
echo "<34>1 2025-08-04T12:00:00Z myhost sloggo - - - Hello, Sloggo" | nc localhost 6514
```

3. Access the application:
   - Frontend: [http://localhost:8080/](http://localhost:8080/)

A health check endpoint is available at [http://localhost:8080/api/health](http://localhost:8080/api/health).

### Testing

To run the backend tests:

```bash
make test
```

## Environment Variables

The following environment variables can be used to configure the application:

- `SLOGGO_LISTENERS`: Comma-separated list of listeners to enable (default: `tcp,udp`).
- `SLOGGO_UDP_PORT`: Port for the UDP Syslog listener (default: `5514`).
- `SLOGGO_TCP_PORT`: Port for the TCP Syslog listener (default: `6514`).
- `SLOGGO_API_PORT`: Port for the API (default: `8080`).
- `SLOGGO_LOG_RETENTION_MINUTES`: Duration in minutes to keep logs before deletion (default: `43200` - 30 days).

## What Sloggo is

- RFC 5424 log ingestion over TCP and UDP
- Real-time search, filtering, and tailing
- Lightweight and resource-efficient single process with zero config
- Can ingests up to 2 million rows/sec in short bursts
- Clean UI built with [data-table-filters](https://github.com/openstatusHQ/data-table-filters)

## What Sloggo is not

- A replacement for full-fledged log management systems like ELK, Loki, or Datadog
- A high availability or redundancy solution
- A logging solution for critical or sensitive data
- A tool for long-term log storage or analysis
- A production-ready solution (yet)

## Why Sloggo?

Slug + log + Go.

> 🐌🤷 Some slugs and snails shoot [love darts](https://en.wikipedia.org/wiki/Love_dart) made of calcium into each other before mating.

## Credits

- [OpenStatus](https://github.com/openstatusHQ) for the incredible [data-table-filters](https://github.com/openstatusHQ/data-table-filters) React components.
- [Leo Di Donato](https://github.com/leodido) for his colossal work on [go-syslog](https://github.com/leodido/go-syslog).

## Contributing

Contributions are welcome! Please fork the repository and submit a pull request with your changes.

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.
