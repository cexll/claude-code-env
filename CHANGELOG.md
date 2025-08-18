# Changelog

All notable changes to this project will be documented in this file.

## [0.2.0] - 2025-08-18

### Added
- Per-environment API key env var selection via new `api_key_env` field in config.
- `cce list` now shows `Key Var: ...` for each environment.
- New CLI flag `--key-var` (and `-k`) to override the API key env var name for a single run.
- `cce version` / `--version` / `-V` to print CLI version.

### Changed
- Relaxed API key validation to be provider-agnostic (length and safety only).

### Docs
- Updated README and README_zh with `api_key_env` usage, examples, and list output.
- Added PRD.md describing the design and requirements for this change.

