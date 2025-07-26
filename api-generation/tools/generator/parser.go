package main

import (
	"fmt"
	"strings"

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
	// Get all actor types
	actorTypes := p.getActorTypes()
	if len(actorTypes) == 0 {
		return fmt.Errorf("no actor types found in OpenAPI specification")
	}

	// Group methods by actor type
	actorMethodsMap := make(map[string][]Method)

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

			// Find which actor type this operation belongs to
			var operationActorType string
			if op.Tags != nil {
				for _, tag := range op.Tags {
					if strings.HasPrefix(tag, "ActorType:") {
						operationActorType = strings.TrimPrefix(tag, "ActorType:")
						break
					}
				}
			}

			if operationActorType == "" {
				continue // Skip operations without actor type
			}

			// Extract method details
			method, err := p.extractMethodFromOperation(op, httpMethod, path)
			if err != nil {
				return fmt.Errorf("failed to extract method from operation %s %s: %v", httpMethod, path, err)
			}

			actorMethodsMap[operationActorType] = append(actorMethodsMap[operationActorType], *method)
		}
	}

	// Create actor interfaces
	for _, actorType := range actorTypes {
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
	// For Dapr actors, extract method name from path (e.g., /{actorId}/method/get -> get)
	methodName := extractMethodNameFromPath(path)
	if methodName == "" {
		return nil, fmt.Errorf("failed to extract method name from path '%s': path must follow pattern '/{actorId}/method/{methodName}'", path)
	}

	// Capitalize the method name for Go interface (exported method)
	methodName = capitalizeFirst(methodName)

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
	}

	return method, nil
}

// extractMethodNameFromPath extracts the method name from Dapr actor path
// e.g., "/{actorId}/method/get" -> "get"
func (p *OpenAPIParser) extractMethodNameFromPath(path string) string {
	// Look for pattern: /{actorId}/method/{methodName}
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "method" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// getOperationComment extracts comment from operation summary/description
func (p *OpenAPIParser) getOperationComment(op *openapi3.Operation) string {
	if op.Summary != "" {
		return op.Summary
	}
	if op.Description != "" {
		// Use first line of description if multi-line
		lines := strings.Split(strings.TrimSpace(op.Description), "\n")
		return strings.TrimSpace(lines[0])
	}
	return "Generated method from OpenAPI operation"
}

// extractRequestType extracts the request type name from request body
func (p *OpenAPIParser) extractRequestType(requestBody *openapi3.RequestBody) string {
	if requestBody.Content == nil {
		return ""
	}

	// Look for JSON content
	if jsonContent := requestBody.Content.Get("application/json"); jsonContent != nil {
		if jsonContent.Schema != nil && jsonContent.Schema.Ref != "" {
			// Extract type name from $ref
			parts := strings.Split(jsonContent.Schema.Ref, "/")
			if len(parts) > 0 {
				return parts[len(parts)-1]
			}
		}
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

// getActorTypes extracts all actor types from OpenAPI spec
func (p *OpenAPIParser) getActorTypes() []string {
	actorTypeSet := make(map[string]bool)

	// Extract from tags in operations (e.g., "ActorType:CounterActor")
	for _, pathItem := range p.doc.Paths.Map() {
		operations := []*openapi3.Operation{
			pathItem.Get, pathItem.Post, pathItem.Put, pathItem.Delete, pathItem.Patch,
		}

		for _, op := range operations {
			if op == nil || op.Tags == nil {
				continue
			}

			for _, tag := range op.Tags {
				if strings.HasPrefix(tag, "ActorType:") {
					actorType := strings.TrimPrefix(tag, "ActorType:")
					if actorType != "" {
						actorTypeSet[actorType] = true
					}
				}
			}
		}
	}

	// Convert set to slice
	var actorTypes []string
	for actorType := range actorTypeSet {
		actorTypes = append(actorTypes, actorType)
	}

	// Fallback if no actor types found
	if len(actorTypes) == 0 {
		if p.doc.Info != nil && p.doc.Info.Title != "" {
			title := p.doc.Info.Title
			// Remove common suffixes
			for _, suffix := range []string{" API", " Service", " Interface"} {
				if strings.HasSuffix(title, suffix) {
					title = strings.TrimSuffix(title, suffix)
					break
				}
			}
			// Convert to PascalCase
			actorTypes = append(actorTypes, strings.ReplaceAll(title, " ", ""))
		} else {
			actorTypes = append(actorTypes, "Actor")
		}
	}

	return actorTypes
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
func (p *OpenAPIParser) categorizeTypesIntoActors(model *GenerationModel, allTypes TypeDefinitions) error {
	// Create a map to track which types are used by which actors
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
	
	// Also analyze type dependencies - if a type references another type, 
	// the referenced type should be shared if the referencing type is used by multiple actors
	typeDependencies := make(map[string][]string) // type -> []referenced_types
	for _, structType := range allTypes.Structs {
		for _, field := range structType.Fields {
			// Extract referenced type from field type (handle arrays and pointers)
			fieldType := field.Type
			fieldType = strings.TrimPrefix(fieldType, "[]")
			fieldType = strings.TrimPrefix(fieldType, "*")
			
			// Check if this is a custom type (not a built-in Go type)
			if p.isCustomTypeInDefinitions(fieldType, allTypes) {
				typeDependencies[structType.Name] = append(typeDependencies[structType.Name], fieldType)
			}
		}
	}
	
	// Propagate usage from dependent types
	for parentType, dependencies := range typeDependencies {
		if parentUsage, exists := typeUsage[parentType]; exists {
			for _, depType := range dependencies {
				if depUsage, exists := typeUsage[depType]; exists {
					// Copy usage from parent to dependency
					for actor, used := range parentUsage {
						if used {
							depUsage[actor] = true
						}
					}
				}
			}
		}
	}

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
	
	// Assign struct types to actors or shared collections
	for _, structType := range allTypes.Structs {
		usedByActors := typeUsage[structType.Name]
		actorCount := len(usedByActors)
		
		if actorCount > 1 {
			// Used by multiple actors - make it shared
			model.SharedTypes.Structs = append(model.SharedTypes.Structs, structType)
		} else if actorCount == 1 {
			// Used by single actor - assign it directly to that actor
			for actorType := range usedByActors {
				// Find the actor and add the type to it
				for i, actor := range model.Actors {
					if actor.ActorType == actorType {
						model.Actors[i].Types.Structs = append(model.Actors[i].Types.Structs, structType)
						break
					}
				}
			}
		} else {
			// Not used by any actor - default to shared for safety
			model.SharedTypes.Structs = append(model.SharedTypes.Structs, structType)
		}
	}

	// Assign type aliases to actors or shared collections
	for _, aliasType := range allTypes.Aliases {
		usedByActors := typeUsage[aliasType.Name]
		actorCount := len(usedByActors)
		
		if actorCount > 1 {
			// Used by multiple actors - make it shared
			model.SharedTypes.Aliases = append(model.SharedTypes.Aliases, aliasType)
		} else if actorCount == 1 {
			// Used by single actor - assign it directly to that actor
			for actorType := range usedByActors {
				// Find the actor and add the type to it
				for i, actor := range model.Actors {
					if actor.ActorType == actorType {
						model.Actors[i].Types.Aliases = append(model.Actors[i].Types.Aliases, aliasType)
						break
					}
				}
			}
		} else {
			// Not used by any actor - type aliases are often reusable, default to shared
			model.SharedTypes.Aliases = append(model.SharedTypes.Aliases, aliasType)
		}
	}
	
	return nil
}