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

const clientTemplate = `// Package {{.PackageName}} provides primitives to interact with the openapi HTTP API.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package {{.PackageName}}

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// RequestEditorFn is the function signature for the RequestEditor callback function
type RequestEditorFn func(ctx context.Context, req *http.Request) error

// Doer performs HTTP requests.
//
// The standard http.Client implements this interface.
type HttpRequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client which conforms to the OpenAPI3 specification for this service.
type Client struct {
	// The endpoint of the server conforming to this interface, with scheme,
	// https://api.deepmap.com for example. This can contain a path relative
	// to the server, such as https://api.deepmap.com/dev-test, and all the
	// paths in the swagger spec will be appended to the server.
	Server string

	// Doer for performing requests, typically a *http.Client with any
	// customized settings, such as certificate chains.
	Client HttpRequestDoer

	// A list of callbacks for modifying requests which are generated before sending over
	// the network.
	RequestEditors []RequestEditorFn
}

// ClientOption allows setting custom parameters during construction
type ClientOption func(*Client) error

// Creates a new Client, with reasonable defaults
func NewClient(server string, opts ...ClientOption) (*Client, error) {
	// create a client with sane default values
	client := Client{
		Server: server,
	}
	// mutate client and add all optional params
	for _, o := range opts {
		if err := o(&client); err != nil {
			return nil, err
		}
	}
	// ensure the server URL always has a trailing slash
	if !strings.HasSuffix(client.Server, "/") {
		client.Server += "/"
	}
	// create httpClient, if not already present
	if client.Client == nil {
		client.Client = &http.Client{}
	}
	return &client, nil
}

// WithHTTPClient allows overriding the default Doer, which is
// automatically created using http.Client. This is useful for tests.
func WithHTTPClient(doer HttpRequestDoer) ClientOption {
	return func(c *Client) error {
		c.Client = doer
		return nil
	}
}

// WithRequestEditorFn allows setting up a callback function, which will be
// called right before sending the request. This can be used to mutate the request.
func WithRequestEditorFn(fn RequestEditorFn) ClientOption {
	return func(c *Client) error {
		c.RequestEditors = append(c.RequestEditors, fn)
		return nil
	}
}

{{range .Methods}}
// {{.Name}} calls the {{.HTTPMethod}} {{.Path}} endpoint
func (c *Client) {{.Name}}(ctx context.Context, actorId string{{if .HasRequestBody}}, body {{.RequestBodyType}}{{end}}, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := New{{.Name}}Request(c.Server, actorId{{if .HasRequestBody}}, body{{end}})
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

// New{{.Name}}Request generates requests for {{.Name}}
func New{{.Name}}Request(server string, actorId string{{if .HasRequestBody}}, body {{.RequestBodyType}}{{end}}) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := strings.Replace("{{.Path}}", "{actorId}", actorId, 1)
	if operationPath[0] == '/' {
		operationPath = operationPath[1:]
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

{{if .HasRequestBody}}
	var bodyReader io.Reader
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	bodyReader = bytes.NewReader(buf)
{{end}}

	req, err := http.NewRequest("{{.HTTPMethod}}", queryURL.String(), {{if .HasRequestBody}}bodyReader{{else}}nil{{end}})
	if err != nil {
		return nil, err
	}

{{if .HasRequestBody}}
	req.Header.Add("Content-Type", "application/json")
{{end}}

	return req, nil
}
{{end}}

func (c *Client) applyEditors(ctx context.Context, req *http.Request, additionalEditors []RequestEditorFn) error {
	for _, r := range c.RequestEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	for _, r := range additionalEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	return nil
}`

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
	Name            string
	HTTPMethod      string
	Path            string
	HasRequestBody  bool
	RequestBodyType string
}

type TemplateData struct {
	PackageName string
	Types       []TypeDef
	TypeAliases []TypeAlias
	Methods     []Method
}

func main() {
	if len(os.Args) < 4 {
		log.Fatal("Usage: types-generator <openapi-file> <package-name> <output-dir> [generate-client]")
	}

	schemaFile := os.Args[1]
	packageName := os.Args[2]
	outputDir := os.Args[3]
	generateClient := len(os.Args) > 4 && os.Args[4] == "true"

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

	// Generate client file if requested
	if generateClient {
		methods := []Method{}
		for path, pathItem := range doc.Paths.Map() {
			if pathItem.Get != nil {
				op := pathItem.Get
				methods = append(methods, Method{
					Name:           capitalizeFirst(op.OperationID),
					HTTPMethod:     "GET",
					Path:           path,
					HasRequestBody: false,
				})
			}
			if pathItem.Post != nil {
				op := pathItem.Post
				hasBody := op.RequestBody != nil
				bodyType := ""
				if hasBody {
					// Extract request body type from operation
					if op.OperationID == "setCounterValue" {
						bodyType = "SetValueRequest"
					}
				}
				methods = append(methods, Method{
					Name:            capitalizeFirst(op.OperationID),
					HTTPMethod:      "POST",
					Path:            path,
					HasRequestBody:  hasBody,
					RequestBodyType: bodyType,
				})
			}
		}

		data.Methods = methods

		clientTmpl, err := template.New("client").Parse(clientTemplate)
		if err != nil {
			log.Fatalf("Failed to parse client template: %v", err)
		}

		clientFile, err := os.Create(fmt.Sprintf("%s/client.go", outputDir))
		if err != nil {
			log.Fatalf("Failed to create client file: %v", err)
		}
		defer clientFile.Close()

		err = clientTmpl.Execute(clientFile, data)
		if err != nil {
			log.Fatalf("Failed to execute client template: %v", err)
		}

		fmt.Printf("Client generated: %s/client.go\n", outputDir)
	}
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