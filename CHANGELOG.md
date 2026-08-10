# Changelog

## [0.1.2] — 2026-08-10

### Added

- Advertise `settings` capability so admin-ui discovers SettingsProvider without ListAll probing.

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0/).

## [Unreleased]

## [0.1.1] — 2026-08-10

### Added

- `RegisterSettings` / `SettingsUpdater` for live `data_dir` (`VALIDATE_DATA_DIR`)
- Pin `core` / contracts / `sdk/go/module` to **v0.5.2**

## [0.1.0] — 2026-08-09

### Added

- JSON Schema (`json:…`) and regex (`regex:…`) input validation sidecar.
- Named schema files under `VALIDATE_DATA_DIR` with path-traversal rejection.
- gRPC listen default `:9660` (`VALIDATE_GRPC_ADDR`).
