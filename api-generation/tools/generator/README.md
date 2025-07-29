# Templates for External Dapr Actor Generator

This directory contains the Go templates used by the external Docker-based Dapr actor generator (`ghcr.io/shogotsuneto/dapr-actor-gen:v0.0.1`).

## Templates

The following templates are mounted into the Docker container at `/root/templates/`:

- `actor_types.tmpl` - Generates type definitions for each actor package
- `factory.tmpl` - Generates actor factory functions
- `interface.tmpl` - Generates actor interface definitions  
- `types.tmpl` - Generates shared type definitions

## Usage

These templates are automatically used by the external Docker generator and do not need to be invoked directly. The generation scripts in `../scripts/` handle mounting these templates into the Docker container.

## Architecture

The external Docker generator:
1. Reads OpenAPI specifications
2. Parses them into an intermediate model
3. Uses these templates to generate Go actor code
4. Outputs organized actor packages to the specified directory

The internal Go-based generator has been removed in favor of the external Docker-based approach for consistency and reproducibility.