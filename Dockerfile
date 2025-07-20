# Simple runtime stage using binary built locally
FROM alpine:latest

WORKDIR /root/

# Copy the pre-built binary
COPY server .

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./server"]