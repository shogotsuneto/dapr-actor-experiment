# OpenAPI to Dapr Actor Generator

This tool generates Go actor implementations from OpenAPI specifications for Dapr actors.

## Architecture

The generator is now organized into separate, modular components:

### Core Components

1. **Parser (`parser.go`)** - Converts OpenAPI specifications to an intermediate model
2. **Intermediate Model (`model.go`)** - Schema-agnostic data structures representing the target code structure  
3. **Generator (`main.go`)** - Converts the intermediate model to Go code using templates
4. **Utilities (`utils.go`)** - Shared utility functions

### Benefits of Separation

- **Extensibility**: Easy to add support for other schema formats (JSON Schema, AsyncAPI, etc.) by implementing new parsers
- **Testability**: Each component can be tested independently
- **Maintainability**: Clear separation of concerns between parsing, modeling, and code generation
- **Reusability**: The intermediate model can be used by different generators for different target languages

## Usage

```bash
go build -o bin/generator .
./bin/generator <openapi-file> <output-directory>
```

Example:
```bash
./bin/generator ../../schemas/openapi/multi-actors.yaml ./generated
```

## Architecture Diagram

```
OpenAPI Spec → Parser → Intermediate Model → Generator → Go Code
                ↓              ↓               ↓
           parser.go      model.go        main.go + templates
```

## Model Organization

The intermediate model (`model.go`) is organized in a hierarchical structure:

```
GenerationModel (Root) - Main container for all parsed data
├── Actors []ActorInterface - Collection of actor definitions
│   └── ActorInterface - Individual actor definition
│       ├── ActorType: string - Actor type name (e.g., "Counter")
│       ├── InterfaceName: string - Generated interface name (e.g., "CounterActor")
│       ├── InterfaceDesc: string - Actor description from OpenAPI
│       ├── Types TypeDefinitions - Type definitions used ONLY by this actor
│       │   ├── Structs []StructType - Go struct types to be generated
│       │   │   └── StructType - Go struct type definition
│       │   │       ├── Name: string - Struct name (e.g., "CounterState")
│       │   │       ├── Description: string - Documentation comment
│       │   │       └── Fields: []Field - Struct fields
│       │   │           └── Field - Individual struct field
│       │   │               ├── Name: string - Field name
│       │   │               ├── Type: string - Go type (e.g., "int", "string")
│       │   │               ├── JSONTag: string - JSON struct tag
│       │   │               └── Comment: string - Field documentation
│       │   └── Aliases []TypeAlias - Go type aliases to be generated
│       │       └── TypeAlias - Go type alias definition
│       │           ├── Name: string - Alias name
│       │           ├── Description: string - Documentation comment
│       │           ├── AliasTarget: string - Underlying Go type
│       │           └── OriginalName: string - Original OpenAPI name
│       └── Methods: []Method - Actor method definitions
│           └── Method - Individual actor method
│               ├── Name: string - Method name
│               ├── Comment: string - Method documentation
│               ├── HasRequest: bool - Whether method takes parameters
│               ├── RequestType: string - Parameter type name
│               └── ReturnType: string - Return type name
└── SharedTypes TypeDefinitions - Type definitions used by MULTIPLE actors (generated in shared package)
    ├── Structs []StructType - Same structure as above, but for shared types like "AccountEvent"
    └── Aliases []TypeAlias - Same structure as above, but for shared aliases like "ActorId"
```
```

### Key Distinctions

**Data Structures vs Generated Code:**
- `StructType` struct = metadata describing a Go struct to be generated
- Generated Go struct = actual `.go` code created from `StructType` data
- `TypeAlias` struct = metadata describing a Go type alias to be generated  
- Generated Go type alias = actual `type X = Y` code created from `TypeAlias` data

**Actor-Specific vs Shared:**
- **Actor-Specific Types**: Stored in `ActorInterface.Types`, generated in `internal/{actor}/types.go`
- **Shared Types**: Stored in `GenerationModel.SharedTypes`, generated in `internal/shared/types.go`

**Template Data Structures:**
```
Template Data Structures:
├── ActorModel (for individual actor generation)
├── TypesTemplateData (for types.go files) - contains TypeDefinitions
├── InterfaceTemplateData (for interface generation)
├── SingleActorTemplateData (for single actor files)  
└── SharedTypesTemplateData (for shared types package) - contains TypeDefinitions
```

## Files

- `main.go` - Entry point and code generation logic using intermediate model
- `parser.go` - OpenAPI parsing and conversion to intermediate model
- `model.go` - Intermediate data structures independent of source schema format
- `utils.go` - Shared utility functions for parsing and generation
- `generator_test.go` - Tests for the separated architecture
- `templates/` - Go templates for code generation

## Testing

Run tests to verify the parser and generator work correctly:

```bash
go test -v .
```