package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestBasicActorParsing(t *testing.T) {
	// Load the basic actor test OpenAPI spec
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("testdata/basic-actor.yaml")
	if err != nil {
		t.Fatalf("Failed to load basic actor OpenAPI spec: %v", err)
	}

	// Parse the spec to intermediate model
	parser := NewOpenAPIParser(doc)
	model, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse OpenAPI spec: %v", err)
	}

	// Verify that we have exactly one actor
	if len(model.Actors) != 1 {
		t.Errorf("Expected 1 actor, got %d", len(model.Actors))
	}

	// Verify the Test actor
	actor := model.Actors[0]
	if actor.ActorType != "Test" {
		t.Errorf("Expected actor type 'Test', got '%s'", actor.ActorType)
	}

	if len(actor.Methods) != 2 {
		t.Errorf("Expected Test actor to have 2 methods, got %d", len(actor.Methods))
	}

	// Verify methods
	methodNames := make(map[string]bool)
	for _, method := range actor.Methods {
		methodNames[method.Name] = true
	}
	if !methodNames["GetValue"] {
		t.Error("Expected 'GetValue' method not found")
	}
	if !methodNames["SetValue"] {
		t.Error("Expected 'SetValue' method not found")
	}

	// Verify actor-specific types (single actor gets all types it uses)
	if len(actor.Types.Structs) < 2 {
		t.Errorf("Expected at least 2 struct types for TestActor, got %d", len(actor.Types.Structs))
	}

	// Verify specific types exist
	typeNames := make(map[string]bool)
	for _, structType := range actor.Types.Structs {
		typeNames[structType.Name] = true
	}
	if !typeNames["TestState"] {
		t.Error("Expected 'TestState' type not found")
	}
	if !typeNames["SetValueRequest"] {
		t.Error("Expected 'SetValueRequest' type not found")
	}

}

func TestMultiActorTypeDuplication(t *testing.T) {
	// Load the multi-actor test OpenAPI spec
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("testdata/multi-actor.yaml")
	if err != nil {
		t.Fatalf("Failed to load multi-actor OpenAPI spec: %v", err)
	}

	// Parse the spec to intermediate model
	parser := NewOpenAPIParser(doc)
	model, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse OpenAPI spec: %v", err)
	}

	// Verify that we have exactly two actors
	if len(model.Actors) != 2 {
		t.Errorf("Expected 2 actors, got %d", len(model.Actors))
	}

	// Verify actors exist
	actorTypes := make(map[string]*ActorInterface)
	for i, actor := range model.Actors {
		actorTypes[actor.ActorType] = &model.Actors[i]
	}

	counterActor, hasCounter := actorTypes["Counter"]
	calcActor, hasCalc := actorTypes["Calculator"]

	if !hasCounter {
		t.Error("Counter not found in parsed model")
	}
	if !hasCalc {
		t.Error("Calculator not found in parsed model")
	}

	// Verify Counter methods
	if hasCounter && len(counterActor.Methods) != 3 {
		t.Errorf("Expected Counter to have 3 methods, got %d", len(counterActor.Methods))
	}

	// Verify Calculator methods
	if hasCalc && len(calcActor.Methods) != 3 {
		t.Errorf("Expected Calculator to have 3 methods, got %d", len(calcActor.Methods))
	}

	// Verify that types are duplicated in each actor that uses them
	if hasCounter {
		counterTypeNames := make(map[string]bool)
		for _, structType := range counterActor.Types.Structs {
			counterTypeNames[structType.Name] = true
		}
		// CounterState should be actor-specific
		if !counterTypeNames["CounterState"] {
			t.Error("Expected CounterActor-specific type 'CounterState' not found")
		}
		// OperationLog and LogMetadata should now be duplicated in Counter actor
		if !counterTypeNames["OperationLog"] {
			t.Error("Expected type 'OperationLog' in Counter actor not found")
		}
		if !counterTypeNames["LogMetadata"] {
			t.Error("Expected type 'LogMetadata' in Counter actor not found")
		}
	}

	if hasCalc {
		calcTypeNames := make(map[string]bool)
		for _, structType := range calcActor.Types.Structs {
			calcTypeNames[structType.Name] = true
		}
		// Calculator-specific types
		if !calcTypeNames["MathOperation"] {
			t.Error("Expected CalculatorActor-specific type 'MathOperation' not found")
		}
		if !calcTypeNames["OperationResult"] {
			t.Error("Expected CalculatorActor-specific type 'OperationResult' not found")
		}
		// OperationLog and LogMetadata should now be duplicated in Calculator actor too
		if !calcTypeNames["OperationLog"] {
			t.Error("Expected type 'OperationLog' in Calculator actor not found")
		}
		if !calcTypeNames["LogMetadata"] {
			t.Error("Expected type 'LogMetadata' in Calculator actor not found")
		}
	}
}

func TestTypeAliasGeneration(t *testing.T) {
	// Load the type alias test OpenAPI spec
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile("testdata/type-alias.yaml")
	if err != nil {
		t.Fatalf("Failed to load type alias OpenAPI spec: %v", err)
	}

	// Parse the spec to intermediate model
	parser := NewOpenAPIParser(doc)
	model, err := parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse OpenAPI spec: %v", err)
	}

	// Verify that we have exactly one actor
	if len(model.Actors) != 1 {
		t.Errorf("Expected 1 actor, got %d", len(model.Actors))
	}

	actor := model.Actors[0]
	if actor.ActorType != "User" {
		t.Errorf("Expected actor type 'User', got '%s'", actor.ActorType)
	}

	// Verify that type aliases are generated from parameters
	totalAliases := len(actor.Types.Aliases)
	if totalAliases == 0 {
		t.Error("Expected type aliases to be generated from parameters, but found none")
	}

	// Look for specific type aliases that should be generated
	aliasNames := make(map[string]bool)
	for _, alias := range actor.Types.Aliases {
		aliasNames[alias.Name] = true
	}

	// These should be generated from the schema definitions
	expectedAliases := []string{"UserId", "EmailAddress", "UserStatus"}
	for _, expected := range expectedAliases {
		if !aliasNames[expected] {
			t.Errorf("Expected type alias '%s' not found", expected)
		}
	}
}

func TestGeneratorWithTestSpecs(t *testing.T) {
	tests := []struct {
		name     string
		specFile string
	}{
		{"Basic Actor", "testdata/basic-actor.yaml"},
		{"Multi Actor", "testdata/multi-actor.yaml"},
		{"Type Alias", "testdata/type-alias.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Load and parse the OpenAPI spec
			loader := openapi3.NewLoader()
			doc, err := loader.LoadFromFile(tt.specFile)
			if err != nil {
				t.Fatalf("Failed to load OpenAPI spec %s: %v", tt.specFile, err)
			}

			parser := NewOpenAPIParser(doc)
			model, err := parser.Parse()
			if err != nil {
				t.Fatalf("Failed to parse OpenAPI spec: %v", err)
			}

			// Generate code using the intermediate model
			generator := &Generator{}
			outputDir := filepath.Join("test-output", tt.name)
			err = generator.GenerateActorPackages(model, outputDir)
			if err != nil {
				t.Fatalf("Failed to generate actor packages: %v", err)
			}

			// Clean up after test
			defer func() {
				os.RemoveAll(outputDir)
			}()

			t.Logf("Successfully generated actor packages for %s", tt.name)
		})
	}
}