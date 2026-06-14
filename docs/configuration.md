# Configuration

CCL reads `~/.config/cclauncher/config.yaml`. If the file doesn't exist, CCL creates it with all default providers on first run.

## Quick Override

The fastest way to configure CCL is environment variables — no config file editing required:

```bash
export SYNTHETIC_API_KEY="sk-..."   # enables all Synthetic.new providers
export DEEPSEEK_API_KEY="sk-..."    # enables deepseek
```

Set the default provider by editing `cli.defaultProvider` in `~/.config/cclauncher/config.yaml`.

## Config File Location

```bash
~/.config/cclauncher/config.yaml
```

CCL currently uses `~/.config/cclauncher/config.yaml`. The path is built from
your home directory and does not read `XDG_CONFIG_HOME`.

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

The committed sample at `examples/config.yaml.example` is generated from
`internal/config/default-config.yaml`, the same default config embedded into the
binary for first-run setup. Regenerate it with:

```bash
go run ./internal/tools/gen-config-examples
```

The shape is:

```yaml
# ~/.config/cclauncher/config.yaml

# --- Providers ---------------------------------------------------------------
providers:
  <name>:
    baseUrl: "https://api.example.com/anthropic"  # API endpoint (required)
    authRequired: true            # Set false for local/OAuth providers with no key requirement
    authToken: "${MY_API_KEY}"    # Bearer token for most providers
    apiKey: ""                    # Alternative auth field (less common)
    oauthToken: ""                # OAuth token — rarely used; claude handles its own auth
    model: "model-id"             # Primary model (ANTHROPIC_MODEL)
    smallFastModel: "model-id"    # Background/subagent model (ANTHROPIC_SMALL_FAST_MODEL)

# --- CLI ---------------------------------------------------------------------
cli:
  defaultProvider: "zai"         # Picker pre-selection and non-TTY fallback

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
| `authRequired` | `authRequired` | *(none)* | Whether CCL blocks launch when no credential is configured |
| `authToken` / `apiKey` | `authToken`, `apiKey` | `ANTHROPIC_AUTH_TOKEN` | Bearer token; `authToken` takes precedence |
| `oauthToken` | `oauthToken` | `CLAUDE_CODE_OAUTH_TOKEN` | OAuth token for Anthropic providers |
| `model` | `model` | `ANTHROPIC_MODEL`, `ANTHROPIC_DEFAULT_OPUS_MODEL`, `ANTHROPIC_DEFAULT_SONNET_MODEL` | Primary model |
| `smallFastModel` | `smallFastModel` | `ANTHROPIC_SMALL_FAST_MODEL`, `ANTHROPIC_DEFAULT_HAIKU_MODEL`, `CLAUDE_CODE_SUBAGENT_MODEL` | Background and subagent model |

## Inspect the Loaded Config

```bash
ccl providers
```

This prints provider names, resolved models, whether a credential is currently
available, and the default provider marker. Values come from your local config
after `${VAR}` interpolation, default-provider merging, and
`CCL_<PROVIDER>_API_KEY` overrides.

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

The optional `router` block can pin a model slot to a named provider. This is rarely needed — most users skip it.

**Contract — what CCL actually honors.** Only `router.default` changes runtime behavior today. The other slots (`background`, `longContext`, `think`, `webSearch`, `longContextThreshold`) are accepted by the config parser but **not yet consumed** — setting them has no effect. They are reserved for future use; do not rely on them.

```yaml
# router:
#   default: "deepseek:deepseek-v4-pro"   # the ONLY slot CCL acts on today
#   # Reserved (parsed but currently inert — no effect):
#   # background: "deepseek:deepseek-v4-flash"
#   # longContext: "openrouter:deepseek/deepseek-v3.2"
#   # think: "openrouter:anthropic/claude-sonnet-4.5"
#   # webSearch: "zai:glm-4.7"
#   # longContextThreshold: 60000
```

**Verification — the exact form `default` must take.** `router.default` is `"<provider>:<model>"`. The override applies only when all hold:

- the value contains a `:` separating a non-empty provider and a non-empty model;
- `<provider>` equals the provider CCL selected for this launch (`--provider`, the picker, or the non-TTY `cli.defaultProvider` fallback);
- `<model>` is non-empty.

When it applies, CCL sets both the provider's `model` and `smallFastModel` to `<model>` for that launch.

**reject_if — when the entry is silently ignored (no error, no effect):**

- the value has no `:` (e.g. `"deepseek-v4-pro"`) → ignored;
- `<provider>` names a provider other than the one selected this launch → ignored;
- `<model>` is empty (e.g. `"deepseek:"`) → ignored;
- the slot is one of the reserved slots (`background`, `longContext`, `think`, `webSearch`, `longContextThreshold`) → ignored.

There is no validation error for an ignored `router` entry; it simply does nothing.

**Acceptance checklist (minimal):**

- [ ] With `router.default: "deepseek:deepseek-v4-override"` and provider `deepseek` selected, the launched provider's `model` and `smallFastModel` are both `deepseek-v4-override`. (Covered by `TestResolveProvider` → `"router default overrides matching provider model slots"`, `internal/cli/handlers/handlers_test.go`.)
- [ ] With `router.default` targeting a different provider than the one selected, the selected provider's models are unchanged.
- [ ] Setting only a reserved slot (e.g. `background`) leaves all model slots unchanged.

## Transformer Presets (Advanced)

```yaml
providers:
  myprovider:
    baseUrl: "https://api.example.com/anthropic"
    authToken: "${MYPROVIDER_API_KEY}"
    model: "my-model"
    transformer:
      use:
        - clamp-max-tokens-8192
      rules:
        - modelPattern: "my-model"
          addHeaders:
            X-Routing-Hint: "coding"
```

`transformer.use` loads built-in rule presets. CCL expands preset names during
config load, before the launcher decides whether to start the local proxy.
Preset rules run before inline `transformer.rules`, so provider-specific rules
can add final headers or body changes.

| Preset | Behavior |
|--------|----------|
| `clamp-max-tokens-8192` | If a request asks for more than 8192 `max_tokens`, rewrite `max_tokens` to `8192`. |

Unknown preset names reject the config with a provider-scoped error. They are
not ignored.

### Transformer Rule Fields

```yaml
transformer:
  rules:
    - modelPattern: "claude"
      messagePattern: "route-fast"
      tokenRange:
        min: 8193
      setModel: "provider-fast-model"
      setMaxTokens: 8192
      setTemperature: 0.2
      addHeaders:
        X-Routing-Hint: "fast-lane"
      modifyBody:
        top_p: 0.9
```

All non-empty match fields use AND logic. A matching rule may rewrite the JSON
request body and add headers to the upstream request.

### Evidence

- `transformer.use` is part of the config schema.
- CCL starts the local proxy when either `transformer.use` or
  `transformer.rules` is present.
- Preset names are expanded into concrete rules before the proxy receives the
  provider config.
- The proxy applies only concrete `TransformRule` values.

### Verification

```bash
go test ./internal/config ./internal/proxy
go test ./...
```

Useful acceptance checks for a new preset:

- `transformer.use` with the preset name expands into at least one rule.
- unknown preset names fail loudly.
- preset rules run before inline provider rules.
- a provider with only `transformer.use` mutates a matching request in proxy
  tests.

### Reject If

- unknown presets are ignored.
- `transformer.use` starts the proxy but produces no rules.
- presets depend on network state or local files outside the config.
- preset expansion mutates shared registry state.
- docs mention a preset that has no test proving its request mutation.
