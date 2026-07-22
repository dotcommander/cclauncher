# Changelog

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
