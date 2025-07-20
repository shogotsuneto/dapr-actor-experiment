package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/getkin/kin-openapi/openapi3"
)

const typesTemplate = `// Package {{.PackageName}} provides primitives for OpenAPI-based contract validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package {{.PackageName}}

{{range .Types}}
// {{.Name}} {{.Description}}
type {{.Name}} struct {
{{- range .Fields}}
	// {{.Comment}}
	{{.Name}} {{.Type}} ` + "`" + `json:"{{.JSONTag}}"` + "`" + `
{{- end}}
}
{{end}}
{{range .TypeAliases}}
// {{.Name}} defines model for {{.OriginalName}}.
type {{.Name}} = {{.Type}}
{{end}}`

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

type TemplateData struct {
	PackageName string
	Types       []TypeDef
	TypeAliases []TypeAlias
}

func main() {
	if len(os.Args) < 4 {
		log.Fatal("Usage: types-generator <openapi-file> <package-name> <output-dir>")
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

	// Generate types file
	data := TemplateData{
		PackageName: packageName,
		Types:       types,
		TypeAliases: typeAliases,
	}

	tmpl, err := template.New("types").Parse(typesTemplate)
	if err != nil {
		log.Fatalf("Failed to parse types template: %v", err)
	}

	typesFile, err := os.Create(fmt.Sprintf("%s/types.go", outputDir))
	if err != nil {
		log.Fatalf("Failed to create types file: %v", err)
	}
	defer typesFile.Close()

	err = tmpl.Execute(typesFile, data)
	if err != nil {
		log.Fatalf("Failed to execute types template: %v", err)
	}

	fmt.Printf("Types generated: %s/types.go\n", outputDir)
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