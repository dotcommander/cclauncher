## Development Notes

- Skip the ui and web ui feature! Do NOT implement.
- Dotfiles live at `/path/to/project/Library/CloudStorage/Dropbox/.dotfiles/`; do not assume `~/Dropbox/` or `~/dropbox/`.
- The local LLM utilities script is `llm.sh`, not `llms.sh`.
- Justfiles should include a default recipe that prints the menu:
  ```just
  default:
      @just --list
  ```
- Imported `justclaude` notes were consolidated here. The retained image asset is `docs/assets/justclaude-freeze.png`.

## Verification (BLOCKING)

After ANY code change, you MUST run `just install` (not just `go build`). The user runs the symlinked `~/go/bin/ccl` binary — a build without install means the user still sees old behavior. This is the minimum:

1. `just install` — builds and updates the symlink
2. `go test ./...` — scoped to changed packages when possible
3. `ccl --help` — sanity check the installed binary

## Installation & Setup

### Development Commands (justfile)

The project uses `just` as a task runner. Available commands:

| Command | Description |
|---------|-------------|
| `just build` | Build the binary to `./ccl` |
| `just install` | Build and symlink to `~/go/bin/ccl` |
| `just test` | Run all tests |
| `just clean` | Remove build artifacts |
| `just fmt` | Format Go code |
| `just lint` | Run golangci-lint |
| `just run` | Build and run ccl |
| `just dev` | Run directly without building |

### Binary Location
- Use symlink from `~/go/bin/ccl` pointing to `$(pwd)/ccl`
- Do NOT install copies in `~/.local/bin` - it takes precedence over `~/go/bin` in PATH
- After building: `just install` or `ln -sf $(pwd)/ccl ~/go/bin/ccl`

### Adding New Providers

Provider definitions live in YAML, not Go. To add a provider:

1. **Edit `internal/config/default-config.yaml`** (shipped defaults) and `internal/config/testdata/config.yaml` (test fixture):
   ```yaml
   providers:
     myprovider:
       baseUrl: "https://api.example.com/anthropic"
       authToken: "${MYPROVIDER_API_KEY}"
       model: "model-id"
       smallFastModel: "model-id"
   ```

2. **API endpoint format**: must be Anthropic Messages-compatible. Avoid trailing `/v1` if the endpoint adds it automatically.

3. **Auth interpolation**: use `${VAR}` or `${VAR:-default}` in `authToken`. `CCL_<PROVIDER>_API_KEY` env vars override the config value at load time.

4. **Tests**: add the name to the expected-provider slices in `internal/config/providers_test.go` and a BaseURL row in `TestGetProvider_BaseURLs`.

5. **Docs**: add an entry in `docs/providers.md`.

No Go code changes are required for a new provider — the launcher reads the YAML and sets `ANTHROPIC_*` env vars generically.

## API Compatibility Requirements

### Anthropic Messages API Format
Claude Code uses the Anthropic Messages API format, which expects:
- Endpoint: `/v1/messages`
- Headers: `x-api-key`, `anthropic-version`, `content-type`
- Request body with `model`, `messages`, `max_tokens`, etc.

### Compatible Provider Types
1. **Native Anthropic API**: Direct Anthropic endpoints
2. **Anthropic-compatible APIs**: Providers that implement the Messages API format
   - Synthetic.new (`https://api.synthetic.new/anthropic`)
   - Z.ai (`https://api.z.ai/api/anthropic`)

### Common Issues & Fixes

1. **"unknown provider" error**: Rebuild and reinstall after adding provider
2. **404 API route not found**: Check for duplicate `/v1` in BaseURL
3. **Z.ai `/v4/v1/messages` 404**: Use `/api/anthropic` not `/api/paas/v4`

### Environment Variable Overrides

Per-provider API keys can be overridden using environment variables:
```bash
export CCL_<PROVIDER>_API_KEY="your-key"
```

Format: `CCL_<PROVIDER>_API_KEY` where `<PROVIDER>` is the provider name in uppercase.

Priority order:
1. `CCL_<PROVIDER>_API_KEY` environment variable (highest)
2. Config file `authToken` field (with environment variable interpolation)
3. Default value from built-in provider template

### Authentication Validation

CCL validates that the selected provider has authentication configured before launching. If authentication is missing, you'll see:
```
Error: provider 'synthetic' requires authentication. Set SYNTHETIC_API_KEY environment variable or configure in config.yaml
```

This prevents confusing 401/403 errors from the API later.
