# Stage 1: Build the Go binary
FROM golang:1.25-trixie AS go-builder

ARG VERSION=dev
WORKDIR /app

RUN apt-get update && apt-get install -y build-essential

COPY backend/ .

RUN mkdir -p /app/.duckdb

RUN go mod download
RUN CGO_ENABLED=1 \
    go build \
    -ldflags "-s -w -X sloggo/utils.Version=${VERSION}" \
    -o sloggo main.go

# Stage 2: Build the React frontend
FROM node:22-trixie-slim AS frontend-builder

RUN corepack enable && corepack prepare pnpm@latest --activate
WORKDIR /app

COPY frontend/pnpm-lock.yaml frontend/package.json ./
RUN pnpm fetch

COPY frontend/ .
RUN pnpm install --offline
RUN pnpm exec next telemetry disable
RUN pnpm build

# Stage 3: Final runtime image
FROM gcr.io/distroless/cc-debian13 AS runtime

WORKDIR /app

COPY --from=go-builder /app/sloggo /app/sloggo
COPY --from=go-builder /app/.duckdb /app/.duckdb
COPY --from=frontend-builder /app/out /app/public

EXPOSE 8080 5514 6514

CMD ["./sloggo"]
