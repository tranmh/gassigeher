# Dockerfile for Gassigeher SaaS
# Multi-stage build for optimized production image

FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# GOTOOLCHAIN=auto allows Go to download required toolchain version (1.24+)
ENV GOTOOLCHAIN=auto
RUN go mod download

# Copy source
COPY . .

# Build with version info
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X github.com/tranmh/gassigeher/internal/version.Version=${VERSION} -X github.com/tranmh/gassigeher/internal/version.GitCommit=${GIT_COMMIT} -X github.com/tranmh/gassigeher/internal/version.BuildTime=${BUILD_TIME}" \
    -o gassigeher ./cmd/server

# Runtime image
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/gassigeher .

# Note: Frontend and landing pages are embedded in binary via go:embed
# No additional COPY needed

# Create non-root user for security
RUN addgroup -g 1000 gassigeher && \
    adduser -u 1000 -G gassigeher -s /bin/sh -D gassigeher && \
    chown -R gassigeher:gassigeher /app

USER gassigeher

EXPOSE 8080

# Health check for container orchestration
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

CMD ["./gassigeher"]
