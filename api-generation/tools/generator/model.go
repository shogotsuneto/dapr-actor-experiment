package main

// Field represents a struct field in the intermediate model
type Field struct {
	Name    string
	Type    string
	JSONTag string
	Comment string
}

// TypeDef represents a type definition in the intermediate model
// It can be either a struct type or a type alias
type TypeDef struct {
	Name        string
	Description string
	Fields      []Field    // For struct types - empty for type aliases
	AliasTarget string     // For type aliases - empty for struct types
	OriginalName string    // For type aliases - original parameter name
}

// IsAlias returns true if this TypeDef represents a type alias
func (t *TypeDef) IsAlias() bool {
	return t.AliasTarget != ""
}

// IsStruct returns true if this TypeDef represents a struct type
func (t *TypeDef) IsStruct() bool {
	return len(t.Fields) > 0
}

// Method represents an actor method in the intermediate model
type Method struct {
	Name        string
	Comment     string
	HasRequest  bool
	RequestType string
	ReturnType  string
}

// ActorInterface represents an actor interface in the intermediate model
type ActorInterface struct {
	ActorType     string
	InterfaceName string
	InterfaceDesc string
	Methods       []Method
	// Types contains type definitions (both structs and aliases) specific to this actor only
	Types []TypeDef
}

// GenerationModel represents the complete intermediate data structure
// that is independent of any specific schema format (OpenAPI, etc.)
type GenerationModel struct {
	// Actors contains all actor interfaces with their methods and actor-specific types
	Actors []ActorInterface
	// SharedTypes contains types (both structs and aliases) that should be generated in a shared package (used by multiple actors)
	SharedTypes []TypeDef
}

// ActorModel represents a single actor's complete model for generation
type ActorModel struct {
	ActorType       string
	PackageName     string
	Types           []TypeDef
	ActorInterface  ActorInterface
}

// TypesTemplateData represents data for types template generation
type TypesTemplateData struct {
	PackageName string
	Types       []TypeDef
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

// SharedTypesTemplateData represents data for shared types template generation
type SharedTypesTemplateData struct {
	PackageName string
	SharedTypes []TypeDef
}