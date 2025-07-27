package main

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/getkin/kin-openapi/openapi3"
)

// OpenAPIParser handles conversion from OpenAPI specification to intermediate model
type OpenAPIParser struct {
	doc *openapi3.T
}

// NewOpenAPIParser creates a new OpenAPI parser
func NewOpenAPIParser(doc *openapi3.T) *OpenAPIParser {
	return &OpenAPIParser{doc: doc}
}

// Parse converts the OpenAPI specification to an intermediate GenerationModel
func (p *OpenAPIParser) Parse() (*GenerationModel, error) {
	model := &GenerationModel{}

	// Parse actors and their methods first
	if err := p.parseActors(model); err != nil {
		return nil, fmt.Errorf("failed to parse actors: %v", err)
	}

	// Parse types and categorize them into shared vs actor-specific
	if err := p.parseAndCategorizeTypes(model); err != nil {
		return nil, fmt.Errorf("failed to parse and categorize types: %v", err)
	}

	return model, nil
}

// parseAndCategorizeTypes extracts type definitions from OpenAPI components 
// and categorizes them into shared vs actor-specific types
func (p *OpenAPIParser) parseAndCategorizeTypes(model *GenerationModel) error {
	if p.doc.Components == nil || p.doc.Components.Schemas == nil {
		return nil
	}

	// First, parse all types from the OpenAPI spec
	var allStructs []StructType
	var allAliases []TypeAlias

	// Parse struct types and type aliases from schemas
	for name, schemaRef := range p.doc.Components.Schemas {
		schema := schemaRef.Value
		
		// Check if this should be a type alias (simple type without properties or with only basic properties)
		if !schema.Type.Is("object") || schema.Properties == nil || len(schema.Properties) == 0 {
			// This should be a type alias
			goType := getGoType(schema)
			allAliases = append(allAliases, TypeAlias{
				Name:         name,
				Description:  schema.Description,
				AliasTarget:  goType,
				OriginalName: name,
			})
		} else if schema.Type.Is("object") && schema.Properties != nil {
			// Generate struct type
			fields := []Field{}
			for propName, propRef := range schema.Properties {
				prop := propRef.Value
				
				// Check if this property is a reference to another schema
				var goType string
				if propRef.Ref != "" {
					// Extract referenced type name from $ref
					refParts := strings.Split(propRef.Ref, "/")
					if len(refParts) > 0 {
						goType = refParts[len(refParts)-1]
					} else {
						goType = getGoType(prop)
					}
				} else {
					goType = getGoType(prop)
				}
				
				jsonTag := propName
				if !contains(schema.Required, propName) {
					jsonTag += ",omitempty"
				}
				fields = append(fields, Field{
					Name:    capitalizeFirst(propName),
					Type:    goType,
					JSONTag: jsonTag,
					Comment: prop.Description,
				})
			}
			allStructs = append(allStructs, StructType{
				Name:        name,
				Description: schema.Description,
				Fields:      fields,
			})
		}
	}

	// Parse type aliases from path parameters and components parameters
	for _, pathItem := range p.doc.Paths.Map() {
		for _, param := range pathItem.Parameters {
			p := param.Value
			if p.Schema != nil && p.Schema.Value.Type.Is("string") {
				aliasName := capitalizeFirst(p.Name)
				allAliases = append(allAliases, TypeAlias{
					Name:         aliasName,
					Description:  fmt.Sprintf("defines model for %s", p.Name),
					AliasTarget:  "string",
					OriginalName: p.Name,
				})
			}
		}
	}

	// Also parse type aliases from components.parameters (for referenced parameters)
	if p.doc.Components != nil && p.doc.Components.Parameters != nil {
		for paramName, paramRef := range p.doc.Components.Parameters {
			param := paramRef.Value
			if param.Schema != nil && param.Schema.Value.Type.Is("string") {
				aliasName := capitalizeFirst(paramName)
				allAliases = append(allAliases, TypeAlias{
					Name:         aliasName,
					Description:  fmt.Sprintf("defines model for %s", param.Name),
					AliasTarget:  "string",
					OriginalName: param.Name,
				})
			}
		}
	}

	// Now categorize types based on usage by actors
	allTypes := TypeDefinitions{
		Structs: allStructs,
		Aliases: allAliases,
	}
	return p.categorizeTypesIntoActors(model, allTypes)
}

// parseActors extracts actor interfaces and their methods from OpenAPI paths
func (p *OpenAPIParser) parseActors(model *GenerationModel) error {
	// Group methods by actor type and track discovered actor types
	actorMethodsMap := make(map[string][]Method)
	discoveredActorTypes := make(map[string]bool)

	for path, pathItem := range p.doc.Paths.Map() {
		// Process all HTTP methods in the path
		operations := map[string]*openapi3.Operation{
			"GET":    pathItem.Get,
			"POST":   pathItem.Post,
			"PUT":    pathItem.Put,
			"DELETE": pathItem.Delete,
			"PATCH":  pathItem.Patch,
		}

		for httpMethod, op := range operations {
			if op == nil {
				continue
			}

			// Extract actor type from path pattern
			actorType := p.extractActorTypeFromPath(path)

			if actorType == "" {
				continue // Skip operations without identifiable actor type
			}

			// Track discovered actor types
			discoveredActorTypes[actorType] = true

			// Extract method details
			method, err := p.extractMethodFromOperation(op, httpMethod, path)
			if err != nil {
				return fmt.Errorf("failed to extract method from operation %s %s: %v", httpMethod, path, err)
			}

			actorMethodsMap[actorType] = append(actorMethodsMap[actorType], *method)
		}
	}

	// Fail if no actor types found
	if len(discoveredActorTypes) == 0 {
		return fmt.Errorf("no actor types found in OpenAPI specification - paths must follow pattern: .../{actorType}/{actorId}/method/{methodName}")
	}

	// Create actor interfaces
	for actorType := range discoveredActorTypes {
		methods := actorMethodsMap[actorType]
		if len(methods) == 0 {
			continue // Skip actor types with no methods
		}

		interfaceName := actorType + "API"
		interfaceDesc := fmt.Sprintf("defines the interface that must be implemented to satisfy the OpenAPI schema for %s", actorType)

		model.Actors = append(model.Actors, ActorInterface{
			ActorType:     actorType,
			InterfaceName: interfaceName,
			InterfaceDesc: interfaceDesc,
			Methods:       methods,
		})
	}

	return nil
}

// extractMethodFromOperation extracts method information from OpenAPI operation
func (p *OpenAPIParser) extractMethodFromOperation(op *openapi3.Operation, httpMethod, path string) (*Method, error) {
	// For Dapr actors, extract method name from path (e.g., /{actorType}/{actorId}/method/get -> get)
	methodName := p.extractMethodNameFromPath(path)
	if methodName == "" {
		return nil, fmt.Errorf("failed to extract method name from path '%s': path must follow pattern '/{actorType}/{actorId}/method/{methodName}'", path)
	}

	// Validate that method name starts with capital letter (Go exported method requirement)
	if len(methodName) == 0 || !unicode.IsUpper(rune(methodName[0])) {
		return nil, fmt.Errorf("method name '%s' must start with a capital letter (Go exported method requirement) in path '%s'", methodName, path)
	}

	method := &Method{
		Name:       methodName,
		Comment:    getOperationComment(op),
		HasRequest: false,
		ReturnType: "interface{}", // default return type
	}

	// Check if operation has request body
	if op.RequestBody != nil && op.RequestBody.Value != nil {
		method.HasRequest = true
		// Extract request type from schema
		if requestType := extractRequestType(op.RequestBody.Value); requestType != "" {
			method.RequestType = requestType
		}
	}

	// Extract return type from 200 response
	if returnType := p.extractReturnType(op); returnType != "" {
		method.ReturnType = returnType
		method.EmbeddedType = returnType
		// Generate response type name: capitalize method name + "Response"
		method.ResponseType = methodName + "Response"
	}

	return method, nil
}

// extractMethodNameFromPath extracts the method name from Dapr actor path
// e.g., "/CounterActor/{actorId}/method/get" -> "get"
func (p *OpenAPIParser) extractMethodNameFromPath(path string) string {
	// Look for pattern: /{actorType}/{actorId}/method/{methodName}
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "method" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// extractActorTypeFromPath extracts the actor type from Dapr actor path
// Uses relative position from "method" to find the actor type
// e.g., "/CounterActor/{actorId}/method/get" -> "CounterActor"
// e.g., "/actors/CounterActor/{actorId}/method/get" -> "CounterActor"
func (p *OpenAPIParser) extractActorTypeFromPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) < 4 || parts[0] != "" { // paths should start with /
		return ""
	}
	
	// Find the position of "method" in the path
	methodIndex := -1
	for i, part := range parts {
		if part == "method" {
			methodIndex = i
			break
		}
	}
	
	// Actor type should be 2 positions before "method"
	// .../actorType/{actorId}/method/methodName
	if methodIndex >= 2 {
		return parts[methodIndex-2]
	}
	
	return ""
}

// extractReturnType extracts the return type from 200 response
func (p *OpenAPIParser) extractReturnType(op *openapi3.Operation) string {
	if op.Responses == nil {
		return ""
	}

	// Look for 200 response
	response200 := op.Responses.Status(200)
	if response200 == nil || response200.Value == nil || response200.Value.Content == nil {
		return ""
	}

	// Look for JSON content
	if jsonContent := response200.Value.Content.Get("application/json"); jsonContent != nil {
		if jsonContent.Schema != nil {
			// Handle direct $ref
			if jsonContent.Schema.Ref != "" {
				parts := strings.Split(jsonContent.Schema.Ref, "/")
				if len(parts) > 0 {
					return parts[len(parts)-1]
				}
			}
			
			schema := jsonContent.Schema.Value
			if schema != nil {
				// Handle array schemas with items.$ref
				if schema.Type != nil && schema.Type.Is("array") && schema.Items != nil && schema.Items.Ref != "" {
					parts := strings.Split(schema.Items.Ref, "/")
					if len(parts) > 0 {
						return "[]" + parts[len(parts)-1]
					}
				}
			}
		}
	}

	return ""
}

// isCustomType checks if a type name refers to a custom type defined in the model
// isCustomTypeInDefinitions checks if a type name exists in our type definitions
func (p *OpenAPIParser) isCustomTypeInDefinitions(typeName string, types TypeDefinitions) bool {
	// List of Go built-in types that are not custom
	builtinTypes := map[string]bool{
		"string": true, "int": true, "int32": true, "int64": true,
		"float32": true, "float64": true, "bool": true,
		"interface{}": true, "map[string]interface{}": true,
	}
	
	if builtinTypes[typeName] {
		return false
	}
	
	// Check if it's defined in our struct types
	for _, structType := range types.Structs {
		if structType.Name == typeName {
			return true
		}
	}
	
	// Check if it's defined in our type aliases
	for _, aliasType := range types.Aliases {
		if aliasType.Name == typeName {
			return true
		}
	}
	
	return false
}

// categorizeTypesIntoActors analyzes types and assigns them directly to actors or shared collections
// Based on the new organization principle: shared contains fundamental components/domain concepts,
// actor-specific contains only request parameter schemas
func (p *OpenAPIParser) categorizeTypesIntoActors(model *GenerationModel, allTypes TypeDefinitions) error {
	// Initialize actor type collections
	for i := range model.Actors {
		model.Actors[i].Types = TypeDefinitions{
			Structs: []StructType{},
			Aliases: []TypeAlias{},
		}
	}
	model.SharedTypes = TypeDefinitions{
		Structs: []StructType{},
		Aliases: []TypeAlias{},
	}

	// Create a map to track which types are used by which actors for reference
	typeUsage := make(map[string]map[string]bool) // type -> actor -> used
	
	// Initialize usage map for all types (both structs and aliases)
	for _, structType := range allTypes.Structs {
		typeUsage[structType.Name] = make(map[string]bool)
	}
	for _, aliasType := range allTypes.Aliases {
		typeUsage[aliasType.Name] = make(map[string]bool)
	}
	
	// Analyze which actors use which types by examining request/response schemas
	for _, actor := range model.Actors {
		for _, method := range actor.Methods {
			// Track request types
			if method.HasRequest && method.RequestType != "" {
				if _, exists := typeUsage[method.RequestType]; exists {
					typeUsage[method.RequestType][actor.ActorType] = true
				}
			}
			// Track return types (remove pointer/slice prefixes for analysis)
			returnType := method.ReturnType
			returnType = strings.TrimPrefix(returnType, "*")
			returnType = strings.TrimPrefix(returnType, "[]")
			if returnType != "interface{}" && returnType != "" {
				if _, exists := typeUsage[returnType]; exists {
					typeUsage[returnType][actor.ActorType] = true
				}
			}
		}
	}
	
	// New categorization logic: organize based on type purpose rather than usage count
	// Assign struct types based on their semantic role
	for _, structType := range allTypes.Structs {
		if p.isSharedDomainType(structType.Name, structType) {
			// Fundamental domain concepts go to shared
			model.SharedTypes.Structs = append(model.SharedTypes.Structs, structType)
		} else {
			// Operation-specific request/parameter types go to their respective actors
			assigned := false
			usedByActors := typeUsage[structType.Name]
			
			// If used by exactly one actor, assign to that actor
			if len(usedByActors) == 1 {
				for actorType := range usedByActors {
					for i, actor := range model.Actors {
						if actor.ActorType == actorType {
							model.Actors[i].Types.Structs = append(model.Actors[i].Types.Structs, structType)
							assigned = true
							break
						}
					}
					break
				}
			}
			
			// If not assigned or used by multiple actors, default to shared
			if !assigned {
				model.SharedTypes.Structs = append(model.SharedTypes.Structs, structType)
			}
		}
	}

	// All type aliases go to shared (they represent fundamental concepts like ActorId)
	for _, aliasType := range allTypes.Aliases {
		model.SharedTypes.Aliases = append(model.SharedTypes.Aliases, aliasType)
	}
	
	return nil
}

// isSharedDomainType determines if a type represents a fundamental domain concept
// that should be placed in the shared package
func (p *OpenAPIParser) isSharedDomainType(typeName string, structType StructType) bool {
	// State types are fundamental domain concepts - they represent the core data models
	if strings.HasSuffix(typeName, "State") {
		return true
	}
	
	// Event types are fundamental domain concepts - they represent domain events
	if strings.HasSuffix(typeName, "Event") {
		return true
	}
	
	// History types are fundamental domain concepts - they represent aggregate data views
	if strings.HasSuffix(typeName, "History") {
		return true
	}
	
	// Types that contain lists of events or other aggregate data are shared concepts
	for _, field := range structType.Fields {
		if strings.Contains(field.Type, "Event") || strings.Contains(field.Type, "[]") {
			return true
		}
	}
	
	return false
}