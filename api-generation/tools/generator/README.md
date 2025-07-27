# OpenAPI to Dapr Actor Generator

This tool generates Go actor implementations from OpenAPI specifications for Dapr actors.

## Current Architecture

The generator creates a clear separation between shared domain concepts and actor-specific operation parameters:

### Generated Package Structure

```
internal/
├── shared/
│   └── types.go          # Domain concepts (CounterState, BankAccountState, etc.)
├── counter/
│   ├── api.go           # Interface with response type returns
│   ├── types.go         # Request types + Response wrappers  
│   └── counter.go       # Implementation
└── bankaccount/
    ├── api.go           # Interface with response type returns
    ├── types.go         # Request types + Response wrappers
    └── bankaccount.go   # Implementation
```

### Type Organization Strategy

**Shared Package** (`internal/shared/types.go`):
- **State types**: Core data models (e.g., `CounterState`, `BankAccountState`)
- **Event types**: Domain events (e.g., `AccountEvent`)
- **History/aggregate types**: Computed views (e.g., `TransactionHistory`)
- **Type aliases**: Reusable identifiers (e.g., `ActorId`)

**Actor-Specific Packages** (`internal/{actor}/types.go`):
- **Request types**: Operation parameters (e.g., `SetValueRequest`, `DepositRequest`)
- **Response types**: Auto-generated wrappers that embed shared types (e.g., `GetResponse` embeds `shared.CounterState`)

### Response Type Generation

The generator automatically creates response types for each method:
- Method `Get()` returns `*GetResponse` (embeds `shared.CounterState`)  
- Method `Set()` returns `*SetResponse` (embeds `shared.CounterState`)
- This provides clean API boundaries while maintaining indirect references to shared types

### Core Components

1. **Parser (`parser.go`)** - Converts OpenAPI specifications to an intermediate model
2. **Intermediate Model (`model.go`)** - Schema-agnostic data structures representing the target code structure  
3. **Generator (`main.go`)** - Converts the intermediate model to Go code using templates with automatic response type generation
4. **Utilities (`utils.go`)** - Shared utility functions

### Benefits of Current Architecture

- **Clear separation**: Domain concepts in shared, operation parameters in actor packages
- **Type safety**: Auto-generated response wrappers provide compile-time safety
- **Maintainability**: Semantic-based organization makes type purpose immediately clear
- **Extensibility**: Easy to add support for other schema formats by implementing new parsers
- **Testability**: Each component can be tested independently

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
│       │   ├── Structs []StructType - Request types + auto-generated response types
│       │   │   └── StructType - Go struct type definition
│       │   │       ├── Name: string - Struct name (e.g., "SetValueRequest", "GetResponse")
│       │   │       ├── Description: string - Documentation comment
│       │   │       └── Fields: []Field - Struct fields
│       │   │           └── Field - Individual struct field
│       │   │               ├── Name: string - Field name (embedded types use shared type)
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
│               ├── ReturnType: string - Original return type from OpenAPI (fallback)
│               ├── ResponseType: string - Generated response type name (preferred)
│               └── EmbeddedType: string - Shared type that response embeds
└── SharedTypes TypeDefinitions - Domain concepts used by multiple actors
    ├── Structs []StructType - State, event, and aggregate types (e.g., "CounterState", "AccountEvent")
    └── Aliases []TypeAlias - Shared identifiers (e.g., "ActorId")
```

### Key Distinctions

**Data Structures vs Generated Code:**
- `StructType` struct = metadata describing a Go struct to be generated
- Generated Go struct = actual `.go` code created from `StructType` data
- `TypeAlias` struct = metadata describing a Go type alias to be generated  
- Generated Go type alias = actual `type X = Y` code created from `TypeAlias` data

**Actor-Specific vs Shared:**
- **Actor-Specific Types**: Request parameters + response wrappers stored in `ActorInterface.Types`, generated in `internal/{actor}/types.go`
- **Shared Types**: Domain concepts stored in `GenerationModel.SharedTypes`, generated in `internal/shared/types.go`

**Return Type Strategy:**
- **ReturnType**: Original type from OpenAPI spec (fallback for backward compatibility)
- **ResponseType**: Auto-generated wrapper type name (preferred when available) 
- **EmbeddedType**: The shared domain type that the response wrapper embeds

**Template Data Structures:**
```
Template Data Structures:
├── ActorModel (for individual actor generation)
├── TypesTemplateData (for types.go files) - contains TypeDefinitions with response types
├── InterfaceTemplateData (for interface generation)
├── SingleActorTemplateData (for single actor files) - handles shared imports
└── SharedTypesTemplateData (for shared types package) - contains domain TypeDefinitions
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