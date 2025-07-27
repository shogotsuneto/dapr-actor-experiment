package main

import (
	"fmt"
	"sort"
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

	// Parse types and assign them to actors that use them
	if err := p.parseAndCategorizeTypes(model); err != nil {
		return nil, fmt.Errorf("failed to parse and categorize types: %v", err)
	}

	return model, nil
}

// parseAndCategorizeTypes orchestrates the parsing, sorting, and categorization of types
func (p *OpenAPIParser) parseAndCategorizeTypes(model *GenerationModel) error {
	// Parse all types from the OpenAPI spec
	allTypes, err := p.parseTypes()
	if err != nil {
		return err
	}

	// Sort all types for consistent ordering
	p.sortTypes(&allTypes)

	// Categorize types based on usage by actors
	return p.categorizeTypesIntoActors(model, allTypes)
}

// parseTypes extracts type definitions from OpenAPI components
func (p *OpenAPIParser) parseTypes() (TypeDefinitions, error) {
	var allStructs []StructType
	var allAliases []TypeAlias

	if p.doc.Components == nil || p.doc.Components.Schemas == nil {
		return TypeDefinitions{
			Structs: allStructs,
			Aliases: allAliases,
		}, nil
	}

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

	return TypeDefinitions{
		Structs: allStructs,
		Aliases: allAliases,
	}, nil
}

// sortTypes handles all sorting logic for consistent ordering
func (p *OpenAPIParser) sortTypes(types *TypeDefinitions) {
	// Sort all structs by name
	sort.Slice(types.Structs, func(i, j int) bool {
		return types.Structs[i].Name < types.Structs[j].Name
	})

	// Sort fields within each struct by name
	for i := range types.Structs {
		sort.Slice(types.Structs[i].Fields, func(j, k int) bool {
			return types.Structs[i].Fields[j].Name < types.Structs[i].Fields[k].Name
		})
	}

	// Sort all aliases by name
	sort.Slice(types.Aliases, func(i, j int) bool {
		return types.Aliases[i].Name < types.Aliases[j].Name
	})
}

// parseActors orchestrates the parsing, building, sorting, and creation of actor interfaces
func (p *OpenAPIParser) parseActors(model *GenerationModel) error {
	// Extract operations grouped by actor type
	actorOperations, err := p.extractActorOperations()
	if err != nil {
		return err
	}

	// Build methods from operations
	actorMethods, err := p.buildActorMethods(actorOperations)
	if err != nil {
		return err
	}

	// Sort actors and methods for consistent ordering
	p.sortActors(&actorMethods)

	// Build final actor interfaces
	return p.buildActorInterfaces(model, actorMethods)
}

// extractActorOperations extracts and groups operations by actor type from OpenAPI paths
func (p *OpenAPIParser) extractActorOperations() (map[string][]ActorOperation, error) {
	actorOperations := make(map[string][]ActorOperation)
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

			// Store operation for processing
			actorOperations[actorType] = append(actorOperations[actorType], ActorOperation{
				Operation:  op,
				HTTPMethod: httpMethod,
				Path:       path,
			})
		}
	}

	// Fail if no actor types found
	if len(discoveredActorTypes) == 0 {
		return nil, fmt.Errorf("no actor types found in OpenAPI specification - paths must follow pattern: .../{actorType}/{actorId}/method/{methodName}")
	}

	return actorOperations, nil
}

// buildActorMethods builds method definitions from actor operations
func (p *OpenAPIParser) buildActorMethods(actorOperations map[string][]ActorOperation) (map[string][]Method, error) {
	actorMethods := make(map[string][]Method)

	for actorType, operations := range actorOperations {
		var methods []Method

		for _, operation := range operations {
			// Extract method details
			method, err := p.extractMethodFromOperation(operation.Operation, operation.HTTPMethod, operation.Path)
			if err != nil {
				return nil, fmt.Errorf("failed to extract method from operation %s %s: %v", operation.HTTPMethod, operation.Path, err)
			}

			methods = append(methods, *method)
		}

		actorMethods[actorType] = methods
	}

	return actorMethods, nil
}

// sortActors handles all sorting logic for consistent ordering
func (p *OpenAPIParser) sortActors(actorMethods *map[string][]Method) {
	// Sort methods within each actor by name
	for actorType := range *actorMethods {
		methods := (*actorMethods)[actorType]
		sort.Slice(methods, func(i, j int) bool {
			return methods[i].Name < methods[j].Name
		})
		(*actorMethods)[actorType] = methods
	}
}

// buildActorInterfaces creates the final ActorInterface structs
func (p *OpenAPIParser) buildActorInterfaces(model *GenerationModel, actorMethods map[string][]Method) error {
	for actorType, methods := range actorMethods {
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

	// Sort actors by type name for consistent ordering
	sort.Slice(model.Actors, func(i, j int) bool {
		return model.Actors[i].ActorType < model.Actors[j].ActorType
	})

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

// categorizeTypesIntoActors analyzes types and assigns them directly to actors that use them
// Each actor gets its own copy of types it uses
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
	// the referenced type should also be included in actors that use the referencing type
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
	
	// Assign struct types directly to each actor that uses them
	for _, structType := range allTypes.Structs {
		usedByActors := typeUsage[structType.Name]
		
		// Assign to each actor that uses this type
		for actorType := range usedByActors {
			// Find the actor and add the type to it
			for i, actor := range model.Actors {
				if actor.ActorType == actorType {
					model.Actors[i].Types.Structs = append(model.Actors[i].Types.Structs, structType)
					break
				}
			}
		}
	}

	// Assign type aliases directly to each actor that uses them
	for _, aliasType := range allTypes.Aliases {
		usedByActors := typeUsage[aliasType.Name]
		
		// Assign to each actor that uses this type
		for actorType := range usedByActors {
			// Find the actor and add the type to it
			for i, actor := range model.Actors {
				if actor.ActorType == actorType {
					model.Actors[i].Types.Aliases = append(model.Actors[i].Types.Aliases, aliasType)
					break
				}
			}
		}
	}
	
	// Sort types within each actor for consistent ordering
	for i := range model.Actors {
		sort.Slice(model.Actors[i].Types.Structs, func(j, k int) bool {
			return model.Actors[i].Types.Structs[j].Name < model.Actors[i].Types.Structs[k].Name
		})
		sort.Slice(model.Actors[i].Types.Aliases, func(j, k int) bool {
			return model.Actors[i].Types.Aliases[j].Name < model.Actors[i].Types.Aliases[k].Name
		})
	}
	
	return nil
}