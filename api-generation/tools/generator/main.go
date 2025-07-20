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
}

type TypesTemplateData struct {
	PackageName string
	Types       []TypeDef
	TypeAliases []TypeAlias
}

type InterfaceTemplateData struct {
	PackageName string
	Methods     []Method
}

func main() {
	if len(os.Args) < 4 {
		log.Fatal("Usage: generator <openapi-file> <package-name> <output-dir>")
	}

	schemaFile := os.Args[1]
	packageName := os.Args[2]
	outputDir := os.Args[3]

	// Load OpenAPI spec
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(schemaFile)
	if err != nil {
		log.Fatalf("Failed to load OpenAPI spec: %v", err)
	}

	// Generate types
	err = generateTypes(doc, packageName, outputDir)
	if err != nil {
		log.Fatalf("Failed to generate types: %v", err)
	}

	// Generate interface
	err = generateInterface(doc, packageName, outputDir)
	if err != nil {
		log.Fatalf("Failed to generate interface: %v", err)
	}

	fmt.Printf("Generated files:\n")
	fmt.Printf("  %s/types.go\n", outputDir)
	fmt.Printf("  %s/interface.go\n", outputDir)
}

func generateTypes(doc *openapi3.T, packageName, outputDir string) error {
	// Parse types from schemas
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

		// Add some standard aliases based on response types
		typeAliases = append(typeAliases, TypeAlias{
			Name:         "BadRequest",
			Type:         "Error",
			OriginalName: "BadRequest",
		})
		typeAliases = append(typeAliases, TypeAlias{
			Name:         "ServerError", 
			Type:         "Error",
			OriginalName: "ServerError",
		})
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

func generateInterface(doc *openapi3.T, packageName, outputDir string) error {
	// Parse methods from paths
	methods := []Method{}
	for _, pathItem := range doc.Paths.Map() {
		if pathItem.Get != nil {
			op := pathItem.Get
			if op.OperationID == "getCounterValue" {
				methods = append(methods, Method{
					Name:       "Get",
					Comment:    "Get current counter value",
					HasRequest: false,
				})
			}
		}
		if pathItem.Post != nil {
			op := pathItem.Post
			switch op.OperationID {
			case "incrementCounter":
				methods = append(methods, Method{
					Name:       "Increment",
					Comment:    "Increment counter by 1",
					HasRequest: false,
				})
			case "decrementCounter":
				methods = append(methods, Method{
					Name:       "Decrement",
					Comment:    "Decrement counter by 1",
					HasRequest: false,
				})
			case "setCounterValue":
				methods = append(methods, Method{
					Name:        "Set",
					Comment:     "Set counter to specific value",
					HasRequest:  true,
					RequestType: "SetValueRequest",
				})
			}
		}
	}

	// Load template from file
	templatePath := getTemplatePath("interface.tmpl")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse interface template: %v", err)
	}

	// Generate interface file
	data := InterfaceTemplateData{
		PackageName: packageName,
		Methods:     methods,
	}

	interfaceFile, err := os.Create(fmt.Sprintf("%s/interface.go", outputDir))
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