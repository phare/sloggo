# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS go-builder

ARG VERSION=dev

ENV CGO_ENABLED=1 \
    CGO_CFLAGS="-D_LARGEFILE64_SOURCE" \
    GOOS=linux \
    GOARCH=amd64

RUN apk update && apk add --no-cache musl-dev gcc build-base
RUN mkdir -p /app/.sqlite

WORKDIR /app
COPY backend/ .

RUN go mod download
RUN go build \
    -ldflags '-linkmode external -extldflags "-static" -X sloggo/utils.Version=${VERSION}' \
    -o sloggo main.go


# Stage 2: Build the React frontend
FROM node:20-slim AS frontend-builder

RUN corepack enable && corepack prepare pnpm@latest --activate

WORKDIR /app

# Copy lockfile and manifest first for better caching
COPY frontend/pnpm-lock.yaml frontend/package.json ./

RUN pnpm fetch

COPY frontend/ .

RUN pnpm install --offline
RUN pnpm exec next telemetry disable
RUN pnpm build


# Stage 3: Final runtime image
FROM scratch

WORKDIR /app
COPY --from=go-builder /app/sloggo /app/sloggo
COPY --from=go-builder /app/.sqlite /app/.sqlite
COPY --from=frontend-builder /app/out /app/public

EXPOSE 8080
EXPOSE 6514
EXPOSE 5514

CMD ["./sloggo"]
