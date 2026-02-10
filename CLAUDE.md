## Development Notes

- Skip the ui and web ui feature! Do NOT implement.

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

When adding a new provider to CCL:

1. **Update `internal/config/providers.go`**:
   - Add provider configuration to the `Providers` map
   - Use `syntheticBaseConfig()` for Synthetic.new providers
   - Use `getEnvOrDefault()` for API keys
   - For HuggingFace models via Synthetic.new, prefix with `hf:` (e.g., `hf:owner/model`)

2. **API Endpoint Format**:
   - All providers must use Anthropic-compatible API endpoints
   - Avoid trailing `/v1` in BaseURL if the API adds it automatically

3. **Update CLI commands in `internal/cli/commands.go`**:
   - Add provider to flag descriptions
   - Update example text

4. **Update documentation**:
   - docs/architecture/config.md: Add to providers table

### Provider Configurations

#### Z.ai
Anthropic-compatible API at `/api/anthropic` endpoint.

```go
"zai": {
    BaseURL:        "https://api.z.ai/api/anthropic",
    AuthToken:      getEnvOrDefault("ZAI_API_KEY", ""),
    Model:          "glm-4.7",
    SmallFastModel: "glm-4.5-air",
}
```

#### Synthetic.new
Anthropic-compatible API at `/anthropic` endpoint. Used by most providers.

```go
syntheticBaseConfig("hf:deepseek-ai/DeepSeek-V3.2")
// Expands to:
// BaseURL:        "https://api.synthetic.new/anthropic"
// AuthToken:      getEnvOrDefault("SYNTHETIC_API_KEY", "")
// Model:          "hf:deepseek-ai/DeepSeek-V3.2"
// SmallFastModel: "hf:deepseek-ai/DeepSeek-V3.2"
```

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
export CCL_SYNTHETIC_API_KEY="your-key"
export CCL_ZAI_API_KEY="your-key"
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
