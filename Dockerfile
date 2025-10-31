# Build stage
FROM golang:1.23-alpine AS builder

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
  -o login-server .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 ffxi && \
  adduser -D -u 1000 -G ffxi ffxi

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/login-server /app/login-server

# Change ownership
RUN chown -R ffxi:ffxi /app

# Switch to non-root user
USER ffxi

# Expose ports for different server roles
# Auth: 54230, Data: 54231, View: 54001
EXPOSE 54230 54231 54001

# Set default entrypoint
ENTRYPOINT ["/app/login-server"]

# Default command (can be overridden)
CMD ["--help"]
