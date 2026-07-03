# Input Validate JSON Schema

JSON Schema and regex based input validation provider for MuxCore.

A gRPC sidecar module that validates JSON payloads against JSON Schema (using `santhosh-tekuri/jsonschema/v5`) or regex patterns. Supports inline schema strings and file-based schema references. Without this module, core has no structured input validation capability.

## Key Features

- **JSON Schema validation** — `json:<inline-schema>` or `json:<schema-name>` (loaded from a schemas directory)
- **Regex validation** — `regex:<pattern>` for string matching
- **Detailed errors** — Returns per-location validation error messages for JSON Schema violations
- **Sanitized output** — Passes back the original data on successful validation

## Configuration

| Env var | Default | Description |
|---------|---------|-------------|
| `VALIDATE_GRPC_ADDR` | `:9660` | gRPC listen address |
| `VALIDATE_DATA_DIR` | `./schemas` | Directory containing JSON Schema files |
| `MUXCORE_MODULE_ID` | `input-validate-jsonschema` | Module identity |

## Usage

```bash
export MUXCORE_GRPC_INSECURE=true
input-validate-jsonschema
```

## Contract

Implements `InputValidator` (`pkg/contracts`). Registers capability: `input.validate`.
