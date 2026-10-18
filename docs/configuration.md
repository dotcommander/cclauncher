# Configuration

CCL reads `~/.config/cclauncher/config.yaml`. If the file doesn't exist, CCL creates it with all default providers on first run.

## Quick Override

The fastest way to configure CCL is environment variables — no config file editing required:

```bash
export SYNTHETIC_API_KEY="sk-..."
export SYNTHETIC_MODEL="your-model-id"
export SYNTHETIC_SMALL_MODEL="$SYNTHETIC_MODEL"
```

Shipped providers do not select model IDs. Set `<PROVIDER>_MODEL` and
`<PROVIDER>_SMALL_MODEL`, or edit `model` and `smallFastModel` in your config.
A non-Anthropic provider needs an explicit primary model to launch. The optional
small/fast slot is left unset unless you choose it; configure it too when your
endpoint needs a specific model for background/subagent requests. Native `claude`
uses Claude Code's model selection without a CCL override.

Existing explicit model values are preserved. Updating CCL does not rewrite your
user config or remove previously saved model choices. Environment variables only
apply where the config contains their interpolation templates.

CCL always remembers the provider and startup model after a session exits successfully
(exit status 0), including `--provider` and `--model` selections. The next launch uses
that pair, and the picker pre-selects it. Failed/canceled sessions and help, version,
or administrative commands leave the previous default unchanged. In-session `/model`
switches are managed by Claude Code and are not captured by CCL.

The pair is stored as `cli.defaultProvider`, `cli.defaultModel`, and
`cli.defaultSmallFastModel` in `~/.config/cclauncher/config.yaml`. Provider credentials
and model interpolation templates are not rewritten. Explicit `--model` and matching
`router.default` overrides still take precedence. To choose a new model from a
provider's config/environment after one has been remembered, clear the saved model
fields or pass `--model`.

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
    model: "${MY_MODEL}"                   # no model fallback
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
    model: "${MY_MODEL}"          # Explicit primary model (required except native Anthropic)
    smallFastModel: "${MY_SMALL_MODEL}" # Optional explicit background/subagent model
    maxContextTokens: 0          # Real context size for a custom model; 0 leaves defaults

# --- CLI ---------------------------------------------------------------------
cli:
  defaultProvider: "zai"         # Last successful provider; picker/non-TTY default
  defaultModel: ""              # Last successful startup model
  defaultSmallFastModel: ""     # Matching background/subagent model

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
| `maxContextTokens` | `maxContextTokens` | `CLAUDE_CODE_MAX_CONTEXT_TOKENS` | Real custom-model context size; 0 leaves Claude Code defaults |

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
#   default: "deepseek:deepseek-v4.1-flash"   # the ONLY slot CCL acts on today
#   # Reserved (parsed but currently inert — no effect):
#   # background: "deepseek:deepseek-v4.1-flash"
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

- the value has no `:` (e.g. `"deepseek-v4.1-flash"`) → ignored;
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

`transformer.use` loads built-in rule presets. CCL preserves preset references during config load and resolves a separate deep copy
at launch, before deciding whether to start the local proxy.
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

## Scalar interpolation and persistence

CCL parses YAML before filling omitted fields from shipped defaults, then interpolates
scalar values once. Keys are literal. Replacement punctuation, newlines and embedded
`${VAR}` text remain data; they cannot introduce YAML fields or another substitution.
Nested expressions in source templates are rejected. `${VAR:-}` supports an empty
fallback. String fields remain strings; declared numeric and boolean fields are
converted according to their schema. Explicit empty values and `false` are preserved.
Existing providers inherit omitted non-secret endpoint/model-template/policy fields;
model templates have no fallback IDs, and explicit model values (including empty
strings) are preserved. Credentials are never filled from a shipped provider template.

Credential overrides uppercase the provider name and replace hyphens with underscores.
`CCL_MY_PROVIDER_API_KEY` takes precedence over legacy `CCL_MY-PROVIDER_API_KEY`.
An explicitly present empty override clears credentials too. Overrides set `authToken`
and clear competing `apiKey` and `oauthToken` values.

`doctor.probeConcurrency` defaults to `4` in shipped YAML. Set `1` for sequential
probes. Omission inherits the shipped setting; nonpositive values are rejected.

Saving keeps preset references declarative. General YAML comments and environment
source templates are not preserved by `Save`; remembering the successful selection
updates only CLI defaults and preserves them. Atomic writes follow legitimate symlinks, preserve target permissions,
and sync the replacement and containing directory. A failure after replacement reports
that the new file is already in place. Dangling symlinks are refused.

Provider endpoints must be absolute HTTP(S) URLs; local HTTP is supported. Native
Anthropic can omit a model for direct Claude login. Other endpoints require an
explicit primary model; missing models are rejected before launch. Transformed native launches need a
configured upstream key. `responseFormat` is rejected with provider/rule context.
Exact `"*"` matches any model/message; other patterns match substrings. Message patterns
match serialized raw request JSON, including roles and metadata. Counting requests
support model/header/body edits but never add generation settings.
