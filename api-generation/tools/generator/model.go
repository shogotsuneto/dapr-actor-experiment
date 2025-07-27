package main

import "github.com/getkin/kin-openapi/openapi3"

// Field represents a struct field in the intermediate model
type Field struct {
	Name    string
	Type    string
	JSONTag string
	Comment string
}

// StructType represents a struct type definition in the intermediate model
type StructType struct {
	Name        string
	Description string
	Fields      []Field
}

// TypeAlias represents a type alias definition in the intermediate model
type TypeAlias struct {
	Name         string
	Description  string
	AliasTarget  string
	OriginalName string // For type aliases - original parameter name
}

// TypeDefinitions represents a collection of type definitions
type TypeDefinitions struct {
	Structs []StructType
	Aliases []TypeAlias
}

// Method represents an actor method in the intermediate model
type Method struct {
	Name        string
	Comment     string
	HasRequest  bool
	RequestType string
	ReturnType  string
}

// ActorOperation represents an OpenAPI operation grouped by actor type
type ActorOperation struct {
	Operation  *openapi3.Operation
	HTTPMethod string
	Path       string
}

// ActorInterface represents an actor interface in the intermediate model
type ActorInterface struct {
	ActorType     string
	InterfaceName string
	InterfaceDesc string
	Methods       []Method
	// Types contains type definitions specific to this actor only
	Types TypeDefinitions
}

// GenerationModel represents the complete intermediate data structure
// that is independent of any specific schema format (OpenAPI, etc.)
type GenerationModel struct {
	// Actors contains all actor interfaces with their methods and actor-specific types
	Actors []ActorInterface
}

// ActorModel represents a single actor's complete model for generation
type ActorModel struct {
	ActorType       string
	PackageName     string
	Types           TypeDefinitions
	ActorInterface  ActorInterface
}

// TypesTemplateData represents data for types template generation
type TypesTemplateData struct {
	PackageName string
	Types       TypeDefinitions
}

// InterfaceTemplateData represents data for interface template generation
type InterfaceTemplateData struct {
	PackageName string
	Actors      []ActorInterface
}

// SingleActorTemplateData represents data for single actor template generation
type SingleActorTemplateData struct {
	PackageName string
	Actor       ActorInterface
}