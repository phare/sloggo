# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS go-builder

ENV CGO_ENABLED=1 \
    CGO_CFLAGS="-D_LARGEFILE64_SOURCE" \
    GOOS=linux \
    GOARCH=amd64

RUN apk update && apk add --no-cache musl-dev gcc build-base

WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o sloggo main.go

# Stage 2: Build the React frontend
FROM node:24-alpine AS frontend-builder

WORKDIR /app
COPY frontend/ .
RUN npm install
RUN npm run build

# Stage 3: Create the final slim image
FROM alpine:latest

RUN apk add --no-cache sqlite libc6-compat

WORKDIR /app
COPY --from=go-builder /app/sloggo /app/sloggo
COPY --from=frontend-builder /app/.next /app/public

EXPOSE 8080
EXPOSE 6514
EXPOSE 514

CMD ["./sloggo"]
