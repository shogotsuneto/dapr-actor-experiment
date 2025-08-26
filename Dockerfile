# Build stage
FROM golang:1.25 AS builder

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binaries
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o projector ./cmd/projector  
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o query-server ./cmd/query-server

# Runtime stage
FROM alpine:latest

# Install wget for health checks
RUN apk --no-cache add wget

WORKDIR /bin/

# Copy the binaries from builder stage
COPY --from=builder /app/server .
COPY --from=builder /app/projector .
COPY --from=builder /app/query-server .

# Expose ports
EXPOSE 8080 8081

# Default command (can be overridden)
CMD ["./server"]