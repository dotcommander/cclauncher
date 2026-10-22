# Changelog

## v0.5.6 (2026-10-22)

### Fixed

- Abort canceled or interrupted provider pickers without launching Claude; report cancellation with exit status 130.
- Restore configured model slots for explicit primary aliases and use effective models in doctor checks.
- Preserve YAML null semantics, inherit omitted provider credentials, and save successful selections into empty configuration documents.
- Shut down supervised Claude sessions on SIGHUP, including children that ignore graceful termination.
- Preserve proxy query parameters, User-Agent and repeated beta headers; return JSON route errors and correctly log JSON model names.

### Changed

- Correct documentation for supervised launches, remembered defaults, model aliases, credentials and environment overrides.
- Partial provider entries inherit shipped credential templates unless explicitly cleared with empty or null values. Clear or replace credentials when changing an endpoint.

## v0.5.3 (2026-10-04)

### Added

- Validate and normalize provider configuration, credentials, and request transformations with clearer diagnostics.
- Support terminal-aware provider selection, cancellable provider health checks, and bounded release checks.
- Preserve Claude terminal and cancellation behavior while proxying Anthropic Messages and token-count requests.

### Changed

- Document configuration, provider, proxy, terminal, and update behavior for the current implementation.
- Resolve transformer presets per launch without mutating shared preset state.

## v0.5.2 (2026-07-22)

### Fixed

- Configure Kong's compact tree help layout so command summaries and flags render in their intended sections.

## v0.5.1 (2026-07-14)

### Fixed

- Complete the Kong command migration while preserving local, config-free help and Claude passthrough behavior.
- Restore the styled, context-aware provider picker with Huh.
- Report the tagged module version in binaries installed with `go install`.
- Route doctor and update output through the configured command streams.

### Changed

- Document direct Go build, test, install, and doctor workflows.
- Update Lipgloss to v2.0.5.
