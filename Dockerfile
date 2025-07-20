# Build stage
FROM golang:1.24 AS builder

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Runtime stage
FROM alpine:latest

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/server .

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./server"]