# Multi-stage build for xk6-milvus
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /build

# Install xk6 latest
RUN go install go.k6.io/xk6/cmd/xk6@latest

# Copy source code (including vendor/)
COPY . .

# Build k6 with xk6-milvus extension using vendored dependencies
RUN xk6 build \
    --with github.com/mmga-lab/xk6-milvus=. \
    --output /k6

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Copy k6 binary from builder
COPY --from=builder /k6 /usr/bin/k6

# Create directory for scripts
RUN mkdir -p /scripts

WORKDIR /scripts

# Set k6 as entrypoint
ENTRYPOINT ["k6"]

# Default command
CMD ["version"]
