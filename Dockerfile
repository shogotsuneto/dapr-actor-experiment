# Simple runtime stage using binary built locally
FROM alpine:latest

WORKDIR /root/

# Copy the pre-built binary (build locally first)
COPY bin/server .

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./server"]