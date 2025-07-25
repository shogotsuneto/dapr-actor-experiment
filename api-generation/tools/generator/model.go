package main

// Field represents a struct field in the intermediate model
type Field struct {
	Name    string
	Type    string
	JSONTag string
	Comment string
}

// TypeDef represents a struct type definition in the intermediate model
type TypeDef struct {
	Name        string
	Description string
	Fields      []Field
}

// TypeAlias represents a type alias in the intermediate model
type TypeAlias struct {
	Name         string
	Type         string
	OriginalName string
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
	// Types contains type definitions specific to this actor only
	Types []TypeDef
	// TypeAliases contains type aliases specific to this actor only
	TypeAliases []TypeAlias
}

// GenerationModel represents the complete intermediate data structure
// that is independent of any specific schema format (OpenAPI, etc.)
type GenerationModel struct {
	// Actors contains all actor interfaces with their methods and actor-specific types
	Actors []ActorInterface
	// SharedTypes contains types that should be generated in a shared package (used by multiple actors)
	SharedTypes []TypeDef
	// SharedTypeAliases contains type aliases that should be generated in a shared package (used by multiple actors)
	SharedTypeAliases []TypeAlias
}

// ActorModel represents a single actor's complete model for generation
type ActorModel struct {
	ActorType     string
	PackageName   string
	Types         []TypeDef
	TypeAliases   []TypeAlias
	ActorInterface ActorInterface
}

// TypesTemplateData represents data for types template generation
type TypesTemplateData struct {
	PackageName string
	Types       []TypeDef
	TypeAliases []TypeAlias
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
	PackageName   string
	SharedTypes   []TypeDef
	SharedAliases []TypeAlias
}