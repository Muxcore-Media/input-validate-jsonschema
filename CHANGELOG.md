# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0/).

## [Unreleased]

## [0.1.0] — 2026-08-09

### Added

- JSON Schema (`json:…`) and regex (`regex:…`) input validation sidecar.
- Named schema files under `VALIDATE_DATA_DIR` with path-traversal rejection.
- gRPC listen default `:9660` (`VALIDATE_GRPC_ADDR`).
