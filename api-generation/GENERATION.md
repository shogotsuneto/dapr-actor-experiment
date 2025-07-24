# Code Generation with Organized Package Structure

This document explains how the code generation process works with the new actor-specific package organization.

## Overview

The code generation process has been enhanced to support the new package structure that organizes code by actor type rather than technical layers.

### Before (Technical Layer Organization)
```
internal/
├── actor/                     # Actor implementations  
│   ├── counter.go
│   └── bankaccount.go
└── generated/openapi/         # Generated OpenAPI code
    ├── counter_actor.go
    ├── bank_account_actor.go
    └── types.go
```

### After (Functional Unit Organization)
```
internal/
├── counteractor/              # Complete counter actor package
│   ├── counter.go            # Implementation
│   ├── generated.go          # Generated interfaces
│   └── types.go              # Actor-specific types
└── bankaccountactor/          # Complete bank account actor package
    ├── bankaccount.go        # Implementation
    ├── generated.go          # Generated interfaces
    └── types.go              # Actor-specific types
```

## How Generation Works

### 1. Initial Generation
The standard OpenAPI generator creates files in the traditional structure:
- `internal/generated/openapi/types.go` - All type definitions
- `internal/generated/openapi/counter_actor.go` - CounterActor interface
- `internal/generated/openapi/bank_account_actor.go` - BankAccountActor interface

### 2. Reorganization
A post-processing script (`api-generation/tools/scripts/reorganize-generated.sh`) automatically:
- Creates actor-specific packages (`counteractor`, `bankaccountactor`)
- Moves interface files to `generated.go` in each actor package
- Updates package names from `generated` to the actor package name
- Copies all types to each actor's `types.go` (since implementations may use any type)
- Cleans up the original generated directory

### 3. Result
Each actor package becomes self-contained with:
- Implementation files (manually written)
- Generated interface contracts (`generated.go`) 
- Type definitions (`types.go`)

## Usage

### Generating Code
Run the standard generation command:
```bash
./api-generation/tools/scripts/generate.sh openapi schemas/openapi/multi-actors.yaml
```

This will:
1. Generate to `internal/generated/openapi` (standard location)
2. Automatically reorganize into actor-specific packages
3. Clean up the temporary generated directory

### Adding New Actor Types
To add a new actor type:
1. Add the actor definition to `schemas/openapi/multi-actors.yaml`
2. Update the `ACTOR_MAPPINGS` in `reorganize-generated.sh`:
   ```bash
   declare -A ACTOR_MAPPINGS=(
       ["CounterActor"]="counteractor"
       ["BankAccountActor"]="bankaccountactor"
       ["NewActor"]="newactor"  # Add new mapping
   )
   ```
3. Run the generation script

### Benefits of This Approach
- **Backwards Compatible**: Existing generator tools work unchanged
- **Automatic**: No manual file movement required
- **Consistent**: Guarantees proper package structure every time
- **Maintainable**: Centralized mapping configuration
- **Complete**: Each actor package has all necessary types

## Files Modified
- `api-generation/tools/scripts/generate.sh` - Updated to call reorganization
- `api-generation/tools/scripts/reorganize-generated.sh` - New post-processing script

## Implementation Details
The reorganization script:
- Uses associative arrays to map actor types to package names
- Processes each actor type independently  
- Maintains all generated comments and metadata
- Preserves type completeness by copying all types to each package
- Handles both interface files and type definitions
- Updates package declarations automatically