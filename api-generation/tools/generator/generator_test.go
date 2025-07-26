package main

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestOpenAPIParser(t *testing.T) {
	// Load the test OpenAPI spec
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("../../schemas/openapi/multi-actors.yaml")
	if err != nil {
		t.Fatalf("Failed to load OpenAPI spec: %v", err)
	}

	// Parse the spec to intermediate model
	parser := NewOpenAPIParser(doc)
	model, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse OpenAPI spec: %v", err)
	}

	// Verify that we have the expected actors
	if len(model.Actors) != 2 {
		t.Errorf("Expected 2 actors, got %d", len(model.Actors))
	}

	// Check for CounterActor
	foundCounter := false
	foundBankAccount := false
	for _, actor := range model.Actors {
		if actor.ActorType == "CounterActor" {
			foundCounter = true
			if len(actor.Methods) != 4 {
				t.Errorf("Expected CounterActor to have 4 methods, got %d", len(actor.Methods))
			}
		}
		if actor.ActorType == "BankAccountActor" {
			foundBankAccount = true
			if len(actor.Methods) < 3 {
				t.Errorf("Expected BankAccountActor to have at least 3 methods, got %d", len(actor.Methods))
			}
		}
	}

	if !foundCounter {
		t.Error("CounterActor not found in parsed model")
	}
	if !foundBankAccount {
		t.Error("BankAccountActor not found in parsed model")
	}

	// Verify that we have type definitions (either shared or actor-specific)
	totalTypes := len(model.SharedTypes.Structs) + len(model.SharedTypes.Aliases)
	for _, actor := range model.Actors {
		totalTypes += len(actor.Types.Structs) + len(actor.Types.Aliases)
	}
	if totalTypes == 0 {
		t.Error("Expected to find type definitions in parsed model")
	}

	// Look for CounterState type (should be in CounterActor's types)
	foundCounterState := false
	for _, actor := range model.Actors {
		if actor.ActorType == "CounterActor" {
			for _, structType := range actor.Types.Structs {
				if structType.Name == "CounterState" {
					foundCounterState = true
					if len(structType.Fields) != 1 {
						t.Errorf("Expected CounterState to have 1 field, got %d", len(structType.Fields))
					}
				}
			}
		}
	}
	// Also check shared types in case it's shared
	for _, structType := range model.SharedTypes.Structs {
		if structType.Name == "CounterState" {
			foundCounterState = true
			if len(structType.Fields) != 1 {
				t.Errorf("Expected CounterState to have 1 field, got %d", len(structType.Fields))
			}
		}
	}
	if !foundCounterState {
		t.Error("CounterState type not found in parsed model")
	}
}

func TestGeneratorWithParsedModel(t *testing.T) {
	// Load and parse the OpenAPI spec
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("../../schemas/openapi/multi-actors.yaml")
	if err != nil {
		t.Fatalf("Failed to load OpenAPI spec: %v", err)
	}

	parser := NewOpenAPIParser(doc)
	model, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse OpenAPI spec: %v", err)
	}

	// Generate code using the intermediate model
	generator := &Generator{}
	outputDir := "test-output"
	err = generator.GenerateActorPackages(model, outputDir)
	if err != nil {
		t.Fatalf("Failed to generate actor packages: %v", err)
	}

	// Clean up
	defer func() {
		// Remove test output directory
		// Using system calls for cleanup since this is a test
	}()

	t.Log("Successfully generated actor packages from parsed model")
}