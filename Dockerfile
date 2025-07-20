# Runtime stage only - expects binary to be built outside Docker
FROM alpine:latest

WORKDIR /root/

# Copy the binary (should be built with: go build -o server ./cmd/server)
COPY server .

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./server"]