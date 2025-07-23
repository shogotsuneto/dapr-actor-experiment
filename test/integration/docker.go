package integration

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"time"
)

// DockerComposeManager manages Docker Compose services for integration tests
type DockerComposeManager struct {
	composeFile string
}

// NewDockerComposeManager creates a new Docker Compose manager
func NewDockerComposeManager(composeFile string) *DockerComposeManager {
	return &DockerComposeManager{
		composeFile: composeFile,
	}
}

// StartServices starts the required Docker Compose services
func (d *DockerComposeManager) StartServices(ctx context.Context) error {
	fmt.Println("🐳 Starting Docker Compose services...")
	
	// Start Redis, placement service, actor service, and Dapr sidecar
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", d.composeFile, "up", "-d", "--build", "redis", "placement", "actor-service", "actor-service-dapr")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start services: %w\nOutput: %s", err, output)
	}

	fmt.Println("📦 Services started, waiting for health checks...")
	
	// Wait for services to be ready
	return d.waitForServices(ctx)
}

// StopServices stops all Docker Compose services
func (d *DockerComposeManager) StopServices(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", d.composeFile, "down")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop services: %w\nOutput: %s", err, output)
	}
	return nil
}

// waitForServices waits for Dapr sidecar and actor service to be ready
func (d *DockerComposeManager) waitForServices(ctx context.Context) error {
	client := &http.Client{Timeout: 5 * time.Second}

	// Wait for Dapr sidecar health check
	if err := d.waitForEndpoint(ctx, client, "http://localhost:3500/v1.0/healthz", "Dapr sidecar"); err != nil {
		return err
	}

	// Wait for actor service health check
	if err := d.waitForEndpoint(ctx, client, "http://localhost:8080/health", "Actor service"); err != nil {
		return err
	}

	return nil
}

// waitForEndpoint waits for a specific HTTP endpoint to become available
func (d *DockerComposeManager) waitForEndpoint(ctx context.Context, client *http.Client, url, serviceName string) error {
	maxRetries := 60 // Increased to 60 retries
	retryInterval := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			fmt.Printf("✓ %s is ready (attempt %d/%d)\n", serviceName, i+1, maxRetries)
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}

		if i%10 == 0 { // Log every 10 attempts
			fmt.Printf("⏳ Waiting for %s... (attempt %d/%d)\n", serviceName, i+1, maxRetries)
		}

		time.Sleep(retryInterval)
	}

	return fmt.Errorf("%s health check failed after %d retries", serviceName, maxRetries)
}