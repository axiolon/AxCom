# Copyright 2026 Axiolon Labs
# SPDX-License-Identifier: Apache-2.0

# Build stage
FROM golang:1.25.11-alpine AS builder

# Install build dependencies if needed
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application using standard go build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ecom-engine ./cmd/server/main.go && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o migrate ./cmd/migrate/main.go

# Run stage
FROM alpine:3.21.2

# Install ca-certificates and tzdata for secure HTTPS requests and correct timezone operations
RUN apk --no-cache add ca-certificates tzdata

# Create a non-privileged user and group to run the application securely
RUN addgroup -g 10001 -S appgroup && adduser -u 10001 -S appuser -G appgroup

WORKDIR /app

# Copy binaries and migration files from builder with correct ownership
COPY --from=builder --chown=appuser:appgroup /app/ecom-engine .
COPY --from=builder --chown=appuser:appgroup /app/migrate .
COPY --from=builder --chown=appuser:appgroup /app/migrations ./migrations

# Use the non-root user
USER appuser

EXPOSE 8080

CMD ["./ecom-engine"]
