package main

import (
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

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

// getGoType converts OpenAPI schema type to Go type
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

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// contains checks if a slice contains a specific item
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}