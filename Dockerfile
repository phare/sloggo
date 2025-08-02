# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS go-builder

ENV CGO_ENABLED=1 \
    CGO_CFLAGS="-D_LARGEFILE64_SOURCE" \
    GOOS=linux \
    GOARCH=amd64

RUN apk update && apk add --no-cache musl-dev gcc build-base

WORKDIR /app
COPY backend/ .

RUN go mod download
RUN go build -o sloggo main.go

# Stage 2: Build the React frontend
FROM node:24-alpine AS frontend-builder

RUN corepack enable && corepack prepare pnpm@latest --activate

WORKDIR /app

# Copy only dependency files first for better caching
COPY frontend/pnpm-lock.yaml frontend/package.json ./

RUN pnpm fetch --prod

COPY frontend/ .

RUN pnpm install --offline --prod
RUN pnpm build

# Stage 3: Create the final slim image
FROM alpine:latest

RUN apk add --no-cache sqlite libc6-compat

WORKDIR /app
COPY --from=go-builder /app/sloggo /app/sloggo
COPY --from=frontend-builder /app/out /app/public

EXPOSE 8080
EXPOSE 6514
EXPOSE 514

CMD ["./sloggo"]
