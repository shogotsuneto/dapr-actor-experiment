package main

import (
	"fmt"
	"log"
	"os"
	"text/template"

	"github.com/getkin/kin-openapi/openapi3"
)

const interfaceTemplate = `// Package {{.PackageName}} provides primitives for OpenAPI-based contract validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package {{.PackageName}}

import "context"

// CounterActorContract defines the interface that must be implemented
// to satisfy the OpenAPI contract for CounterActor.
// This interface enforces compile-time contract compliance.
type CounterActorContract interface {
{{- range .Methods}}
	// {{.Comment}}
	{{.Name}}(ctx context.Context{{if .HasRequest}}, request {{.RequestType}}{{end}}) (*CounterState, error)
{{- end}}
}
`

type Method struct {
	Name        string
	Comment     string
	HasRequest  bool
	RequestType string
}

type TemplateData struct {
	PackageName string
	Methods     []Method
}

func main() {
	if len(os.Args) != 4 {
		log.Fatal("Usage: generator <openapi-file> <package-name> <output-file>")
	}

	schemaFile := os.Args[1]
	packageName := os.Args[2]
	outputFile := os.Args[3]

	// Load OpenAPI spec
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(schemaFile)
	if err != nil {
		log.Fatalf("Failed to load OpenAPI spec: %v", err)
	}

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

	// Generate interface
	tmpl, err := template.New("interface").Parse(interfaceTemplate)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	file, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	data := TemplateData{
		PackageName: packageName,
		Methods:     methods,
	}

	err = tmpl.Execute(file, data)
	if err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}

	fmt.Printf("Interface generated: %s\n", outputFile)
}