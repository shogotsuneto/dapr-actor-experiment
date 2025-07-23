package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// SnapshotTester provides snapshot testing capabilities for fast test execution
type SnapshotTester struct {
	testName    string
	snapshotDir string
}

// NewSnapshotTester creates a new snapshot tester
func NewSnapshotTester(t *testing.T) *SnapshotTester {
	snapshotDir := filepath.Join(".", "testdata", "snapshots")
	return &SnapshotTester{
		testName:    t.Name(),
		snapshotDir: snapshotDir,
	}
}

// MatchSnapshot compares the given data with a stored snapshot
// If the snapshot doesn't exist, it creates one
// If UPDATE_SNAPSHOTS env var is set, it updates existing snapshots
func (s *SnapshotTester) MatchSnapshot(t *testing.T, name string, data interface{}) {
	// Ensure snapshot directory exists
	if err := os.MkdirAll(s.snapshotDir, 0755); err != nil {
		t.Fatalf("Failed to create snapshot directory: %v", err)
	}

	// Convert data to pretty JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal data to JSON: %v", err)
	}

	snapshotFile := filepath.Join(s.snapshotDir, fmt.Sprintf("%s_%s.json", s.testName, name))

	// Check if we should update snapshots
	updateSnapshots := os.Getenv("UPDATE_SNAPSHOTS") == "true"

	if updateSnapshots {
		// Update/create snapshot
		if err := os.WriteFile(snapshotFile, jsonData, 0644); err != nil {
			t.Fatalf("Failed to write snapshot file: %v", err)
		}
		t.Logf("Updated snapshot: %s", snapshotFile)
		return
	}

	// Try to read existing snapshot
	existingData, err := os.ReadFile(snapshotFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Create new snapshot
			if err := os.WriteFile(snapshotFile, jsonData, 0644); err != nil {
				t.Fatalf("Failed to create snapshot file: %v", err)
			}
			t.Logf("Created new snapshot: %s", snapshotFile)
			return
		}
		t.Fatalf("Failed to read snapshot file: %v", err)
	}

	// Compare snapshots
	if !bytes.Equal(jsonData, existingData) {
		t.Errorf("Snapshot mismatch for %s.\nExpected:\n%s\nActual:\n%s\n\nTo update snapshots, run with UPDATE_SNAPSHOTS=true", 
			name, string(existingData), string(jsonData))
	}
}

// MatchJSONSnapshot compares JSON response data with snapshots
func (s *SnapshotTester) MatchJSONSnapshot(t *testing.T, name string, jsonBytes []byte) {
	var data interface{}
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	s.MatchSnapshot(t, name, data)
}

// LoadSnapshot loads a snapshot for comparison in tests
func (s *SnapshotTester) LoadSnapshot(t *testing.T, name string) []byte {
	snapshotFile := filepath.Join(s.snapshotDir, fmt.Sprintf("%s_%s.json", s.testName, name))
	data, err := os.ReadFile(snapshotFile)
	if err != nil {
		t.Fatalf("Failed to load snapshot %s: %v", snapshotFile, err)
	}
	return data
}

// SnapshotExists checks if a snapshot exists
func (s *SnapshotTester) SnapshotExists(name string) bool {
	snapshotFile := filepath.Join(s.snapshotDir, fmt.Sprintf("%s_%s.json", s.testName, name))
	_, err := os.Stat(snapshotFile)
	return err == nil
}

// WithSnapshotTesting runs a test function with snapshot testing enabled
func WithSnapshotTesting(t *testing.T, testFunc func(*testing.T, *SnapshotTester)) {
	snapshotter := NewSnapshotTester(t)
	testFunc(t, snapshotter)
}

// SnapshotActorResponse creates a snapshot of an actor method response
func SnapshotActorResponse(t *testing.T, snapshotter *SnapshotTester, client *DaprClient, req ActorMethodRequest, snapshotName string) {
	resp, err := client.InvokeActorMethod(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	
	snapshotter.MatchJSONSnapshot(t, snapshotName, resp.Body)
}

// Helper function to create snapshots directory if it doesn't exist
func init() {
	if err := os.MkdirAll("./testdata/snapshots", 0755); err != nil {
		// Ignore error during init - will be handled in tests if needed
	}
}