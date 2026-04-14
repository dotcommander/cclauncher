# Configuration

CCL reads `~/.config/cclauncher/config.yaml`. If the file doesn't exist, CCL creates it with all default providers on first run.

## Quick Override

The fastest way to configure CCL is environment variables — no config file editing required:

```bash
export SYNTHETIC_API_KEY="sk-..."   # enables all Synthetic.new providers
export DEEPSEEK_API_KEY="sk-..."    # enables deepseek
ccl use deepseek                    # persist default provider
```

## Config File Location

```bash
~/.config/cclauncher/config.yaml
```

CCL respects `XDG_CONFIG_HOME` on Linux and `os.UserConfigDir()` on all platforms.

## Environment Variable Interpolation

Config values support `${VAR}` and `${VAR:-default}` syntax. CCL expands these at load time before any provider field is read.

```yaml
providers:
  myprovider:
    authToken: "${MY_API_KEY}"              # empty string if MY_API_KEY is unset
    baseUrl: "${MY_BASE_URL:-https://api.example.com}"  # fallback if unset
    model: "${MY_MODEL:-my-default-model}"
```

| Syntax | Behavior |
|--------|----------|
| `${VAR}` | Replaced with the env var value; empty string if unset |
| `${VAR:-default}` | Replaced with the env var value, or `default` if unset |

Interpolation runs on `baseUrl`, `authToken`, `oauthToken`, `apiKey`, `model`, and `smallFastModel`.

## Per-Provider Key Override

Every provider supports a `CCL_<PROVIDER>_API_KEY` environment variable that overrides the config file value. Use this to test a different key without editing `config.yaml`.

```bash
export CCL_SYNTHETIC_API_KEY="sk-test-key"   # overrides SYNTHETIC_API_KEY for synthetic
export CCL_DEEPSEEK_API_KEY="sk-other"       # overrides deepseek authToken
```

The format is `CCL_` + provider name in uppercase + `_API_KEY`.

**Auth priority order (highest to lowest):**
1. `CCL_<PROVIDER>_API_KEY` environment variable
2. `authToken` in `config.yaml` (after `${VAR}` interpolation)
3. Built-in default (empty — requires you to set an env var or config value)

## Full YAML Schema

```yaml
# ~/.config/cclauncher/config.yaml

# --- Providers ---------------------------------------------------------------
providers:
  <name>:
    baseUrl: "https://api.example.com/anthropic"  # API endpoint (required)
    authToken: "${MY_API_KEY}"    # Bearer token for most providers
    apiKey: ""                    # Alternative auth field (less common)
    oauthToken: ""                # OAuth token — rarely used; claude handles its own auth
    model: "model-id"             # Primary model (ANTHROPIC_MODEL)
    smallFastModel: "model-id"    # Background/subagent model (ANTHROPIC_SMALL_FAST_MODEL)

# --- CLI ---------------------------------------------------------------------
cli:
  defaultProvider: "zai"         # Provider used when --provider is not set

# --- Optimization ------------------------------------------------------------
optimization:
  disableNonessentialTraffic: true   # Sets CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1
  disableAutoupdater: true           # Sets DISABLE_AUTOUPDATER=1
  disableTelemetry: true             # Sets DISABLE_TELEMETRY=1
  disableErrorReporting: true        # Sets DISABLE_ERROR_REPORTING=1
  disableCostWarnings: true          # Sets DISABLE_COST_WARNINGS=1
  apiTimeoutMs: 3000000              # Sets API_TIMEOUT_MS (default: 50 min)
  maxOutputTokens: 200000            # Sets CLAUDE_CODE_MAX_OUTPUT_TOKENS
  nodeMaxOldSpaceSize: 8192          # Sets NODE_OPTIONS=--max-old-space-size=N (MB)
```

## Optimization Settings

The `optimization` block controls environment variables passed to every `claude` invocation. All settings are on by default in the shipped config.

| Setting | Env var set | Default | When to change |
|---------|-------------|---------|----------------|
| `disableNonessentialTraffic` | `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1` | `true` | Disable to re-enable background HTTP calls |
| `disableAutoupdater` | `DISABLE_AUTOUPDATER=1` | `true` | Disable to let `claude` auto-update |
| `disableTelemetry` | `DISABLE_TELEMETRY=1` | `true` | Disable to send telemetry to Anthropic |
| `disableErrorReporting` | `DISABLE_ERROR_REPORTING=1` | `true` | Disable to report errors to Anthropic |
| `disableCostWarnings` | `DISABLE_COST_WARNINGS=1` | `true` | Disable to see per-request cost warnings |
| `apiTimeoutMs` | `API_TIMEOUT_MS` | `3000000` | Lower for faster timeout on slow models |
| `maxOutputTokens` | `CLAUDE_CODE_MAX_OUTPUT_TOKENS` | `200000` | Reduce to limit output length |
| `nodeMaxOldSpaceSize` | `NODE_OPTIONS=--max-old-space-size=N` | `8192` | Increase for very large codebases |

> **Note:** CCL strips any `ANTHROPIC_*` and `CLAUDE_CODE_*` variables from the parent shell before launching `claude`. Without stripping, a stale `ANTHROPIC_BASE_URL` from a previous session would silently route your provider-switched request to the wrong endpoint — and you'd see it as an auth failure, not a misroute. The `DISABLE_*` and `API_TIMEOUT_MS` variables are not stripped — you can still override them from your shell.

## Provider Fields Reference

| Field | Yaml key | Env var set | Purpose |
|-------|----------|-------------|---------|
| `baseUrl` | `baseUrl` | `ANTHROPIC_BASE_URL` | API endpoint |
| `authToken` / `apiKey` | `authToken`, `apiKey` | `ANTHROPIC_AUTH_TOKEN` | Bearer token; `authToken` takes precedence |
| `oauthToken` | `oauthToken` | `CLAUDE_CODE_OAUTH_TOKEN` | OAuth token for Anthropic providers |
| `model` | `model` | `ANTHROPIC_MODEL`, `ANTHROPIC_DEFAULT_OPUS_MODEL`, `ANTHROPIC_DEFAULT_SONNET_MODEL` | Primary model |
| `smallFastModel` | `smallFastModel` | `ANTHROPIC_SMALL_FAST_MODEL`, `ANTHROPIC_DEFAULT_HAIKU_MODEL`, `CLAUDE_CODE_SUBAGENT_MODEL` | Background and subagent model |

## Adding a Custom Provider

You don't need to fork CCL or edit Go code to add a provider. Add an entry to the `providers` map in `config.yaml`:

```yaml
providers:
  myprovider:
    baseUrl: "https://api.myprovider.com/anthropic"
    authToken: "${MYPROVIDER_API_KEY}"
    model: "my-model"
    smallFastModel: "my-fast-model"
```

Then use it immediately:

```bash
export MYPROVIDER_API_KEY="sk-..."
ccl --provider myprovider
```

See [contributing/adding-providers.md](contributing/adding-providers.md) for details on provider types, HuggingFace model prefixes, and transformer rules.

## Router Configuration (Advanced)

The optional `router` block lets you pin specific model slots to named providers. This is rarely needed — most users skip it.

```yaml
# router:
#   default: "deepseek:deepseek-chat"
#   background: "synthetic:hf:zai-org/GLM-4.7"
#   longContext: "openrouter:deepseek/deepseek-v3.2"
#   think: "openrouter:anthropic/claude-sonnet-4.5"
#   longContextThreshold: 50000
```
