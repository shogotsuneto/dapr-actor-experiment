# Build stage
FROM golang:1.24 AS builder

WORKDIR /app

# Copy source code including vendor directory
COPY . .

# Build the binary using vendor dependencies (no network required)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -mod=vendor -o server ./cmd/server

# Runtime stage
FROM alpine:latest

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/server .

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./server"]