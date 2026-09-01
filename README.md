# Input Validate JSON Schema

JSON Schema and regex based input validation provider for MuxCore.

A gRPC sidecar module that validates JSON payloads against JSON Schema (using `santhosh-tekuri/jsonschema/v5`) or regex patterns. Supports inline schema strings and file-based schema references. Without this module, core has no structured input validation capability.

## Key Features

- **JSON Schema validation** — inline `json:{…}` or named `json:<schema-name>` (loads `{VALIDATE_DATA_DIR}/{schema-name}.json`)
- **Named schema safety** — names must match `^[a-z][a-z0-9_.:-]{0,127}$`; `.` / `/` / `\` path forms are rejected; files resolve only under `VALIDATE_DATA_DIR`
- **Regex validation** — `regex:<pattern>` for string matching with a 1s default timeout
- **Detailed errors** — Returns per-location validation error messages for JSON Schema violations
- **Sanitized output** — On success, returns original JSON bytes, or trimmed bytes for regex matches
- **Go client** — `pkg/client` implements `contracts.InputValidator` over gRPC

## How It Works

```
Caller ──FindByCapability("input.validate")──→ core registry
      ──dial module HTTPAddr (:9665)──────────→ input-validate-jsonschema
      ──Validate / SupportedSchemas───────────→ pkg/client (InputValidator)
```

Core discovers the sidecar by capability (`input.validate`) and dials the module's gRPC address from registration (`HTTPAddr`). Use `github.com/Muxcore-Media/input-validate-jsonschema/pkg/client` for the `contracts.InputValidator` adapter.

## Configuration

| Env var | Default | Description |
|---------|---------|-------------|
| `VALIDATE_GRPC_ADDR` | `:9665` | Module gRPC listen address |
| `VALIDATE_DATA_DIR` | `./schemas` | Directory containing JSON Schema files (`{name}.json`) |
| `MUXCORE_GRPC_ADDR` | (required) | Core mesh gRPC address (SDK; or `--muxcore-mesh-addr`) |
| `MUXCORE_MODULE_ID` | `input-validate-jsonschema` | Registration ID override (SDK) |
| `MUXCORE_INSECURE_DISABLE_TLS` | unset | Set to `true` to disable TLS to core (dev only) |

### Size limits

| Limit | Value |
|-------|-------|
| Max request data | 4 MiB |
| Max inline schema string | 4 MiB |
| Regex match timeout | 1s (when caller context has no deadline) |

Oversize requests return `Valid=false` with an error message (not a gRPC error). gRPC `MaxRecvMsgSize` matches the payload cap.

### MVP stack

Enable in `_mvp/run-host.sh` with `MVP_ENABLE_INPUT_VALIDATE=1`. Schemas land under `$DATA/schemas/` (port **9665**, see `_mvp/PORTS.md`). Example schemas ship in this repo under `schemas/` (`user-create.json`, `user-update.json`).

## Usage

```bash
make build
export MUXCORE_GRPC_ADDR=localhost:9090
export MUXCORE_INSECURE_DISABLE_TLS=true
./input-validate-jsonschema
```

### From Go (mesh client)

```go
import (
    "context"
    validateclient "github.com/Muxcore-Media/input-validate-jsonschema/pkg/client"
    "google.golang.org/grpc"
)

conn, _ := grpc.NewClient("127.0.0.1:9665", grpc.WithTransportCredentials(insecureCreds))
validator := validateclient.New(conn)

result, err := validator.Validate(ctx, []byte(`{"name":"alice","email":"a@b.c"}`), "json:user-create")
schemas := validator.SupportedSchemas()
```

Call `SupportedSchemas()` before `Validate` when probing availability.

## Contract

Provides `InputValidator` over gRPC (`InputValidateService`). Declares capability: `input.validate`. Min core: `0.5.0`.
