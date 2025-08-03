<p align="center"><img src="/sloggo-logo.png" width="300" alt="Sloggo Logo"></p>

<p align="center">
<a href="https://github.com/phare/sloggo/actions/workflows/build.yml"><img src="https://img.shields.io/github/actions/workflow/status/phare/sloggo/build" alt="Build"></a>
<a href="https://github.com/phare/sloggo/tags"><img src="https://img.shields.io/github/v/tag/phare/sloggo" alt="Version"></a>
<a href="https://github.com/phare/sloggo?tab=MIT-1-ov-file#readme"><img src="https://img.shields.io/github/license/phare/sloggo" alt="License"></a>
</p>

# Sloggo

Sloggo is a minimalist logging receiver and viewer tool designed to handle a moderate amount of logs efficiently and provide a simple interface for exploring them. It supports receiving Syslog RFC 5424 logs and storing them in a lightweight SQLite database that fits in a single lightweight Docker container.

> [!WARNING]
> Sloggo does not offer any security layer by default, only use it in a private network, or behind a secure reverse proxy.

The UI is based on OpenStatus' [data table filter](https://github.com/openstatusHQ/data-table-filters) 🫶.

## Getting Started

### Running the Project

1. Clone the repository:

   ```bash
   git clone https://github.com/phare/sloggo.git
   cd sloggo
   ```

2. Build and run the Docker container:

   ```bash
   make dev
   ```

3. Access the application:
   - Backend health check: [http://localhost:8080/health](http://localhost:8080/health)
   - Frontend: [http://localhost:8080/](http://localhost:8080/)

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

## Contributing

Contributions are welcome! Please fork the repository and submit a pull request with your changes.

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.
