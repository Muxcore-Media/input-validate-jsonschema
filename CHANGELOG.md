# Changelog

## [0.1.6] - 2026-10-05

### Changed
- Built on core v0.6.14 / sdk/go/module v0.6.4: unregisters on shutdown and re-registers after core restarts (ADR-0022).

## [0.1.3] — 2026-08-10

### Fixed

- Self-hosted CI (`runs-on: self-hosted`; `go test` without `-race` for laptop runners)

### Changed

- `muxcore.json` / `Info()` align `minCoreVersion` and contract pin to **0.5.0** (matches `go.mod` core **v0.5.8**)

## [0.1.2] — 2026-08-10

### Added

- Advertise `settings` capability so admin-ui discovers SettingsProvider without ListAll probing.

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0/).

## [0.1.5] - 2026-10-05


### Changed

- Reported version comes from muxcore.json (ADR-0021); built on core v0.6.12 / sdk/go/module v0.6.3 (mesh enrollment, ADR-0017).

## [0.1.4] - 2026-10-05

### Changed
- CI runs on GitHub-hosted runners from the umbrella template; retired-origin workflows removed.
- Dependencies resolve from published GitHub tags (no filesystem `replace`); requires core v0.6.0.

### Changed

- Inbound gRPC serves TLS by default (`grpc.Creds`); auto-generates mesh-local certs under `~/.muxcore/tls/input-validate-jsonschema` unless `VALIDATE_TLS_*` / `MUXCORE_TLS_*` are set.
- Default bind address is loopback `127.0.0.1:9665` (`VALIDATE_GRPC_ADDR` override unchanged).
- `MUXCORE_INSECURE_DISABLE_TLS` / `MUXCORE_GRPC_INSECURE` disable inbound TLS for local development.

## [0.1.1] — 2026-08-10

### Added

- `RegisterSettings` / `SettingsUpdater` for live `data_dir` (`VALIDATE_DATA_DIR`)
- Pin `core` / contracts / `sdk/go/module` to **v0.5.2**

## [0.1.0] — 2026-08-09

### Added

- JSON Schema (`json:…`) and regex (`regex:…`) input validation sidecar.
- Named schema files under `VALIDATE_DATA_DIR` with path-traversal rejection.
- gRPC listen default `:9665` (`VALIDATE_GRPC_ADDR`).
