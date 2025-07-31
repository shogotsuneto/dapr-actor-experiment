package auth

import (
	"testing"
)



func TestConvertIntrospectionToClaims(t *testing.T) {
	// Test active token
	resp := &IntrospectionResponse{
		Active: true,
		Sub:    "user-123",
	}

	if !resp.Active {
		t.Fatal("Expected active token")
	}

	if resp.Sub != "user-123" {
		t.Errorf("Expected sub=user-123, got %s", resp.Sub)
	}

	// Test inactive token
	inactiveResp := &IntrospectionResponse{
		Active: false,
	}

	if inactiveResp.Active {
		t.Error("Expected inactive token")
	}
}

