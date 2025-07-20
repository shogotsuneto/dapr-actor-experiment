# Build stage
FROM golang:1.24-alpine AS builder

# Install ca-certificates for HTTPS downloads
RUN apk --no-cache add ca-certificates git

WORKDIR /app

# Copy go mod and sum files
COPY ../go.mod ../go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the client application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o client .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/client .

# Run the binary
CMD ["./client"]