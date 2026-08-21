# Input Validate JSON Schema

JSON Schema and regex based input validation provider for MuxCore.

A gRPC sidecar module that validates JSON payloads against JSON Schema (using `santhosh-tekuri/jsonschema/v5`) or regex patterns. Supports inline schema strings and file-based schema references. Without this module, core has no structured input validation capability.

## Key Features

- **JSON Schema validation** — inline `json:{…}` or named `json:<schema-name>` (loads `{VALIDATE_DATA_DIR}/{schema-name}.json`)
- **Named schema safety** — names must match `^[a-z][a-z0-9_.:-]{0,127}$`; `.` / `/` / `\` path forms are rejected; files resolve only under `VALIDATE_DATA_DIR`
- **Regex validation** — `regex:<pattern>` for string matching (unchanged)
- **Detailed errors** — Returns per-location validation error messages for JSON Schema violations
- **Sanitized output** — On success, returns original JSON bytes, or trimmed bytes for regex matches

## Configuration

| Env var | Default | Description |
|---------|---------|-------------|
| `VALIDATE_GRPC_ADDR` | `:9665` | Module gRPC listen address |
| `VALIDATE_DATA_DIR` | `./schemas` | Directory containing JSON Schema files (`{name}.json`) |
| `MUXCORE_GRPC_ADDR` | (required) | Core mesh gRPC address (SDK; or `--muxcore-mesh-addr`) |
| `MUXCORE_MODULE_ID` | `input-validate-jsonschema` | Registration ID override (SDK) |
| `MUXCORE_INSECURE_DISABLE_TLS` | unset | Set to `true` to disable TLS to core (dev only) |

## Usage

```bash
make build
export MUXCORE_GRPC_ADDR=localhost:9090
export MUXCORE_INSECURE_DISABLE_TLS=true
./input-validate-jsonschema
```

## Contract

Provides `InputValidator` over gRPC (`InputValidateService`). Declares capability: `input.validate`. Min core: `0.4.0`.
