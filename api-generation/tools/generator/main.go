package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/getkin/kin-openapi/openapi3"
)

type Field struct {
	Name    string
	Type    string
	JSONTag string
	Comment string
}

type TypeDef struct {
	Name        string
	Description string
	Fields      []Field
}

type TypeAlias struct {
	Name         string
	Type         string
	OriginalName string
}

type Method struct {
	Name        string
	Comment     string
	HasRequest  bool
	RequestType string
	ReturnType  string
}

type ActorInterface struct {
	ActorType     string
	InterfaceName string
	InterfaceDesc string
	Methods       []Method
}

type TypesTemplateData struct {
	PackageName string
	Types       []TypeDef
	TypeAliases []TypeAlias
}

type InterfaceTemplateData struct {
	PackageName string
	Actors      []ActorInterface
}

type SingleActorTemplateData struct {
	PackageName string
	Actor       ActorInterface
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: generator <openapi-file> <base-output-dir>")
	}

	schemaFile := os.Args[1]
	baseOutputDir := os.Args[2]

	// Load OpenAPI spec
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(schemaFile)
	if err != nil {
		log.Fatalf("Failed to load OpenAPI spec: %v", err)
	}

	// Generate actor-specific packages
	err = generateActorPackages(doc, baseOutputDir)
	if err != nil {
		log.Fatalf("Failed to generate actor packages: %v", err)
	}
}

func generateActorPackages(doc *openapi3.T, baseOutputDir string) error {
	// Get all actor types
	actorTypes := getActorTypes(doc)
	
	if len(actorTypes) == 0 {
		return fmt.Errorf("no actor types found in OpenAPI specification")
	}

	// Group methods by actor type
	actorMethodsMap := make(map[string][]Method)
	
	for path, pathItem := range doc.Paths.Map() {
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
			method, err := extractMethodFromOperation(op, httpMethod, path)
			if err != nil {
				return fmt.Errorf("failed to extract method from operation %s %s: %v", httpMethod, path, err)
			}
			
			actorMethodsMap[operationActorType] = append(actorMethodsMap[operationActorType], *method)
		}
	}

	// Generate package for each actor type
	for _, actorType := range actorTypes {
		methods := actorMethodsMap[actorType]
		if len(methods) == 0 {
			continue // Skip actor types with no methods
		}

		// Create actor-specific package name and directory
		packageName := strings.ToLower(actorType)
		if !strings.HasSuffix(packageName, "actor") {
			packageName += "actor"
		}
		
		outputDir := filepath.Join(baseOutputDir, packageName)
		
		// Create output directory
		err := os.MkdirAll(outputDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create output directory %s: %v", outputDir, err)
		}

		// Generate types for this actor
		err = generateActorTypes(doc, packageName, outputDir, actorType)
		if err != nil {
			return fmt.Errorf("failed to generate types for %s: %v", actorType, err)
		}

		// Generate interface for this actor
		err = generateActorInterface(doc, packageName, outputDir, actorType, methods)
		if err != nil {
			return fmt.Errorf("failed to generate interface for %s: %v", actorType, err)
		}

		fmt.Printf("Generated actor package: %s\n", outputDir)
		fmt.Printf("  %s/types.go\n", outputDir)
		fmt.Printf("  %s/generated.go\n", outputDir)
	}

	return nil
}

func generateActorTypes(doc *openapi3.T, packageName, outputDir, actorType string) error {
	// Parse types from schemas - for now, include all types in each actor package
	// In the future, we could filter types based on usage by each actor
	types := []TypeDef{}
	typeAliases := []TypeAlias{}

	if doc.Components != nil && doc.Components.Schemas != nil {
		for name, schemaRef := range doc.Components.Schemas {
			schema := schemaRef.Value
			if schema.Type.Is("object") && schema.Properties != nil {
				// Generate struct type
				fields := []Field{}
				for propName, propRef := range schema.Properties {
					prop := propRef.Value
					goType := getGoType(prop)
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
				types = append(types, TypeDef{
					Name:        name,
					Description: schema.Description,
					Fields:      fields,
				})
			}
		}

		// Generate type aliases for parameter types
		for _, pathItem := range doc.Paths.Map() {
			for _, param := range pathItem.Parameters {
				p := param.Value
				if p.Schema != nil && p.Schema.Value.Type.Is("string") {
					aliasName := capitalizeFirst(p.Name)
					typeAliases = append(typeAliases, TypeAlias{
						Name:         aliasName,
						Type:         "string",
						OriginalName: p.Name,
					})
				}
			}
		}
	}

	// Load template from file
	templatePath := getTemplatePath("types.tmpl")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse types template: %v", err)
	}

	// Generate types file
	data := TypesTemplateData{
		PackageName: packageName,
		Types:       types,
		TypeAliases: typeAliases,
	}

	typesFile, err := os.Create(fmt.Sprintf("%s/types.go", outputDir))
	if err != nil {
		return fmt.Errorf("failed to create types file: %v", err)
	}
	defer typesFile.Close()

	err = tmpl.Execute(typesFile, data)
	if err != nil {
		return fmt.Errorf("failed to execute types template: %v", err)
	}

	return nil
}

func generateActorInterface(doc *openapi3.T, packageName, outputDir, actorType string, methods []Method) error {
	// Load template from file
	templatePath := getTemplatePath("interface.tmpl")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse interface template: %v", err)
	}

	interfaceName := actorType + "API"
	interfaceDesc := fmt.Sprintf("defines the interface that must be implemented to satisfy the OpenAPI schema for %s", actorType)
	
	actor := ActorInterface{
		ActorType:     actorType,
		InterfaceName: interfaceName,
		InterfaceDesc: interfaceDesc,
		Methods:       methods,
	}

	// Generate interface file for this actor
	data := SingleActorTemplateData{
		PackageName: packageName,
		Actor:       actor,
	}

	// Use generated.go as filename for consistency with existing structure
	interfaceFile, err := os.Create(filepath.Join(outputDir, "generated.go"))
	if err != nil {
		return fmt.Errorf("failed to create interface file: %v", err)
	}
	defer interfaceFile.Close()

	err = tmpl.Execute(interfaceFile, data)
	if err != nil {
		return fmt.Errorf("failed to execute interface template: %v", err)
	}

	return nil
}

// extractMethodFromOperation extracts method information from OpenAPI operation
func extractMethodFromOperation(op *openapi3.Operation, httpMethod, path string) (*Method, error) {
	// For Dapr actors, extract method name from path (e.g., /{actorId}/method/get -> get)
	methodName := extractMethodNameFromPath(path)
	if methodName == "" {
		return nil, fmt.Errorf("failed to extract method name from path '%s': path must follow pattern '/{actorId}/method/{methodName}'", path)
	}
	
	// Capitalize the method name for Go interface (exported method)
	methodName = strings.Title(methodName)

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
	if returnType := extractReturnType(op); returnType != "" {
		method.ReturnType = returnType
	}

	return method, nil
}

// extractMethodNameFromPath extracts the method name from Dapr actor path
// e.g., "/{actorId}/method/get" -> "get"
func extractMethodNameFromPath(path string) string {
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
func getOperationComment(op *openapi3.Operation) string {
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
func extractRequestType(requestBody *openapi3.RequestBody) string {
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
func extractReturnType(op *openapi3.Operation) string {
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

// getActorTypes extracts all actor types from OpenAPI spec
func getActorTypes(doc *openapi3.T) []string {
	actorTypeSet := make(map[string]bool)
	
	// Extract from tags in operations (e.g., "ActorType:CounterActor")
	for _, pathItem := range doc.Paths.Map() {
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
		if doc.Info != nil && doc.Info.Title != "" {
			title := doc.Info.Title
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

// getInterfaceName generates interface name from API info
func getInterfaceName(doc *openapi3.T) string {
	if doc.Info != nil && doc.Info.Title != "" {
		// Convert title to PascalCase and add "Contract"
		title := strings.ReplaceAll(doc.Info.Title, " ", "")
		return title + "Contract"
	}
	return "API"
}

// getInterfaceDescription generates interface description from API info
func getInterfaceDescription(doc *openapi3.T) string {
	if doc.Info != nil && doc.Info.Title != "" {
		return fmt.Sprintf("defines the interface that must be implemented to satisfy the OpenAPI schema for %s", doc.Info.Title)
	}
	return "defines the interface that must be implemented to satisfy the OpenAPI schema"
}

// toSnakeCase converts PascalCase to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}



func getTemplatePath(templateName string) string {
	// Get the directory where this binary is located
	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	
	// Look for templates directory relative to the executable
	execDir := filepath.Dir(execPath)
	templatePath := filepath.Join(execDir, "..", "templates", templateName)
	
	// If not found, try relative to current working directory (for development)
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		wd, _ := os.Getwd()
		templatePath = filepath.Join(wd, "templates", templateName)
		
		// If still not found, try relative to the generator source directory
		if _, err := os.Stat(templatePath); os.IsNotExist(err) {
			// Try to find the templates directory in the project structure
			// Walk up from the executable to find api-generation/tools/generator/templates
			currentDir := execDir
			for i := 0; i < 10; i++ { // Limit search depth
				testPath := filepath.Join(currentDir, "generator", "templates", templateName)
				if _, err := os.Stat(testPath); err == nil {
					templatePath = testPath
					break
				}
				testPath = filepath.Join(currentDir, "tools", "generator", "templates", templateName)
				if _, err := os.Stat(testPath); err == nil {
					templatePath = testPath
					break
				}
				testPath = filepath.Join(currentDir, "api-generation", "tools", "generator", "templates", templateName)
				if _, err := os.Stat(testPath); err == nil {
					templatePath = testPath
					break
				}
				currentDir = filepath.Dir(currentDir)
				if currentDir == "/" || currentDir == filepath.Dir(currentDir) {
					break
				}
			}
		}
	}
	
	return templatePath
}

func getGoType(schema *openapi3.Schema) string {
	switch {
	case schema.Type.Is("string"):
		return "string"
	case schema.Type.Is("integer"):
		if schema.Format == "int32" {
			return "int32"
		}
		return "int"
	case schema.Type.Is("number"):
		if schema.Format == "float" {
			return "float32"
		}
		return "float64"
	case schema.Type.Is("boolean"):
		return "bool"
	case schema.Type.Is("array"):
		if schema.Items != nil {
			return "[]" + getGoType(schema.Items.Value)
		}
		return "[]interface{}"
	case schema.Type.Is("object"):
		if schema.AdditionalProperties.Has != nil && *schema.AdditionalProperties.Has {
			return "map[string]interface{}"
		}
		return "interface{}"
	default:
		return "interface{}"
	}
}

func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}