# Adding Providers

Add a provider in `~/.config/cclauncher/config.yaml` — no code changes needed:

```yaml
providers:
  myprovider:
    baseUrl: "https://api.myprovider.com/anthropic"
    authRequired: true
    authToken: "${MYPROVIDER_API_KEY}"
    model: "my-model-name"
    smallFastModel: "my-fast-model"
```

Set the key and use it immediately:

```bash
export MYPROVIDER_API_KEY="your-key"
ccl --provider myprovider
```

## Provider Types

### Anthropic-Compatible (direct)

Most providers implement the Anthropic Messages API at an `/anthropic` path:

```yaml
myprovider:
  baseUrl: "https://api.myprovider.com/anthropic"
  authRequired: true
  authToken: "${MYPROVIDER_API_KEY}"
  model: "model-name"
  smallFastModel: "model-name"
```

> **Warning:** Don't add a trailing `/v1` to `baseUrl`. Many providers append it automatically — a double `/v1/v1/messages` path returns a 404.

### HuggingFace via Synthetic.new

For HuggingFace models hosted on Synthetic.new, prefix the model ID with `hf:`:

```yaml
mymodel:
  baseUrl: "https://api.synthetic.new/anthropic"
  authToken: "${SYNTHETIC_API_KEY}"
  model: "hf:org/model-name"
  smallFastModel: "hf:org/model-name"
```

### Providers Needing Transformer Rules

If a provider requires request modification — model remapping, header injection — add transformer rules. CCL starts a local proxy when transformer rules are present:

```yaml
myprovider:
  baseUrl: "https://api.myprovider.com"
  authToken: "${MYPROVIDER_API_KEY}"
  model: "model-name"
  transformer:
    rules:
      - modelPattern: "*"
        setModel: "actual-model-id"
```

You can also opt into a built-in preset with `transformer.use`:

```yaml
myprovider:
  baseUrl: "https://api.myprovider.com"
  authToken: "${MYPROVIDER_API_KEY}"
  model: "model-name"
  transformer:
    use:
      - clamp-max-tokens-8192
```

Preset rules run before inline `transformer.rules`. Unknown preset names fail
config loading; they are not ignored. See [../configuration.md](../configuration.md#transformer-presets-advanced) for the preset list and acceptance checklist.

Exact `"*"` matches every model or request. Other model patterns are substring
matches. `messagePattern` searches the original raw JSON request, including
metadata, model names, punctuation and JSON escaping; it does not search only
message text. Rules run in order, and later model/token conditions see mutations
from earlier rules. Requests with no matching rule keep their original bytes.

The proxy supports both `/v1/messages` and `/v1/messages/count_tokens`. Counting
requests accept model, header and appropriate body edits, but setters and
`modifyBody` cannot add `max_tokens`, `temperature`, `top_p`, `top_k`,
`stop_sequences` or `stream`. Token-range conditions require `max_tokens` to be
present. `responseFormat` is unsupported and fails configuration validation.

Transformed launches require configured upstream credentials, including native
Anthropic launches. Direct native launches continue to use Claude's own login.
The child receives a fixed local placeholder credential; real credentials stay
in the proxy. Configured authentication overrides incoming or transform header
credentials. A configured `Host` header sets the upstream HTTP Host.

Upstream traffic honors environment proxy settings. Child-only `NO_PROXY` and
`no_proxy` lists preserve existing entries and include localhost, 127.0.0.1 and
::1. The proxy retains upstream errors, request IDs, retry instructions and
Anthropic rate-limit metadata. Local invalid requests return JSON errors with
400, oversized requests with 413 and transport failures with 502.

Claude shares the foreground terminal. Terminal Ctrl-C reaches Claude directly;
controlled parent cancellation uses SIGTERM or caller cancellation, with a
five-second grace period before killing an unresponsive child. Child exit and
signal status survive proxy cleanup. Active streams are canceled on child exit;
cleanup then has an independent five-second budget. Routine proxy events are
logged only at DEBUG.

## Adding to the Default Config

To include a provider in the config template shipped with CCL (so new users get it on first run), edit `internal/config/default-config.yaml`, then regenerate the committed examples:

```bash
go run ./internal/tools/gen-config-examples
```

Open a pull request with the canonical config and generated example changes together.

## Authentication

Override any provider's API key without editing the config file:

```bash
export CCL_MYPROVIDER_API_KEY="override-key"
```

Priority: `CCL_<PROVIDER>_API_KEY` > config `authToken` (with `${VAR}` interpolation) > built-in default.

See [../configuration.md](../configuration.md) for the full auth priority reference.

## Verification

```bash
go build -o ccl ./cmd/ccl
mkdir -p "$(go env GOPATH)/bin"
ln -sf "$(pwd)/ccl" "$(go env GOPATH)/bin/ccl"
ccl providers              # Verify the provider appears in the list
ccl --provider myprovider  # Test launch
```

## Troubleshooting

| Issue | Cause | Fix |
|-------|-------|-----|
| `unknown provider` | Typo in provider name or config not saved | Check spelling in `config.yaml` |
| `404` on API | Duplicate `/v1` in `baseUrl` | Remove trailing `/v1` from `baseUrl` |
| Auth failures | Env var not set or wrong name | Run `echo $MYPROVIDER_API_KEY` to verify |
