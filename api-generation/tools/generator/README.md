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