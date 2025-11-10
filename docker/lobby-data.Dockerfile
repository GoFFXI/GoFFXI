# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build arguments for version info
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
  -ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}" \
  -o lobby-data ./cmd/lobby-data

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 goffxi && \
  adduser -D -u 1000 -G goffxi goffxi

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/lobby-data /app/lobby-data

# Change ownership
RUN chown -R goffxi:goffxi /app

# Switch to non-root user
USER goffxi

# Expose port
EXPOSE 54231

# Set default entrypoint
ENTRYPOINT ["/app/lobby-data"]
