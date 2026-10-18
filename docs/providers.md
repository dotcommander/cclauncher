# Providers

CCL ships with 11 provider endpoint templates across three categories. Models are user-selected, not shipped defaults. CCL sets the correct `ANTHROPIC_*` environment variables and calls `claude` — no proxy or translation layer runs unless you configure transformer rules.

```bash
ccl providers
```

Use the command above to see the providers and models loaded from your current
`~/.config/cclauncher/config.yaml`. The tables below describe the committed
defaults in `internal/config/default-config.yaml`; `examples/config.yaml.example`
is generated from that same canonical file.

For every non-Anthropic provider, set `model` explicitly in your config or set its
`<PROVIDER>_MODEL` variable. Set `smallFastModel` or `<PROVIDER>_SMALL_MODEL` to
choose background/subagent models too. Neither slot has a model fallback. An unset
primary model blocks launch and hides that provider from the interactive picker;
CCL does not assume a server supports default-model routing. Native `claude`
leaves model selection to Claude Code. Existing explicit config values are preserved;
model environment variables apply only where your config uses their templates.

The interactive launch picker hides providers with missing required credentials,
invalid launch configuration, or unreachable localhost/loopback endpoints. Local
checks use unauthenticated HEAD requests, the doctor's five-second probe timeout,
and `doctor.probeConcurrency`; redirects and environment proxies are disabled.
Any HTTP response counts as reachable, including servers that reject HEAD.
Cloud endpoints are not probed. `ccl providers` still lists every configured
provider, and explicit `--provider` launches are unchanged. Eligibility does not
prove credential validity or that the requested model is loaded.

## Cloud (Native APIs)

These providers use their own API endpoints and require separate keys.

### synthetic

```bash
export SYNTHETIC_API_KEY="sk-..."
export SYNTHETIC_MODEL="your-model-id"
export SYNTHETIC_SMALL_MODEL="$SYNTHETIC_MODEL"
ccl --provider synthetic
```

| Field | Value |
|-------|-------|
| Endpoint | `https://api.synthetic.new/anthropic` |
| Model | User-selected (`SYNTHETIC_MODEL`) |
| Small/Fast | User-selected (`SYNTHETIC_SMALL_MODEL`) |
| Env var | `SYNTHETIC_API_KEY` |

Choose a model hosted by Synthetic.new using the variables above or `~/.config/cclauncher/config.yaml`.

### deepseek

```bash
export DEEPSEEK_API_KEY="sk-..."
export DEEPSEEK_MODEL="your-model-id"
export DEEPSEEK_SMALL_MODEL="$DEEPSEEK_MODEL"
ccl --provider deepseek
```

| Field | Value |
|-------|-------|
| Endpoint | `https://api.deepseek.com/anthropic` |
| Model | User-selected (`DEEPSEEK_MODEL`) |
| Small/Fast | User-selected (`DEEPSEEK_SMALL_MODEL`) |
| Env var | `DEEPSEEK_API_KEY` |

### minimax

```bash
export MINIMAX_API_KEY="sk-..."
export MINIMAX_MODEL="your-model-id"
export MINIMAX_SMALL_MODEL="$MINIMAX_MODEL"
ccl --provider minimax
```

| Field | Value |
|-------|-------|
| Endpoint | `https://api.minimax.io/anthropic` |
| Model | User-selected (`MINIMAX_MODEL`) |
| Small/Fast | User-selected (`MINIMAX_SMALL_MODEL`) |
| Env var | `MINIMAX_API_KEY` |

### zai

```bash
export ZAI_API_KEY="sk-..."
export ZAI_MODEL="your-model-id"
export ZAI_SMALL_MODEL="$ZAI_MODEL"
ccl --provider zai
```

| Field | Value |
|-------|-------|
| Endpoint | `https://api.z.ai/api/anthropic` |
| Model | User-selected (`ZAI_MODEL`) |
| Small/Fast | User-selected (`ZAI_SMALL_MODEL`) |
| Env var | `ZAI_API_KEY` |

`zai` supports model overrides without editing the config file:

```bash
export ZAI_MODEL="your-model-id"
export ZAI_SMALL_MODEL="$ZAI_MODEL"
```

### openrouter

```bash
export OPENROUTER_API_KEY="sk-or-..."
export OPENROUTER_MODEL="your-model-id"
export OPENROUTER_SMALL_MODEL="$OPENROUTER_MODEL"
ccl --provider openrouter
```

| Field | Value |
|-------|-------|
| Endpoint | `https://openrouter.ai/api` |
| Model | User-selected (`OPENROUTER_MODEL`) |
| Small/Fast | User-selected (`OPENROUTER_SMALL_MODEL`) |
| Env var | `OPENROUTER_API_KEY` |

OpenRouter exposes an Anthropic Messages-compatible route at `/api` (its "Anthropic Skin"). Swap the model to any [OpenRouter slug](https://openrouter.ai/models) — e.g. `anthropic/claude-sonnet-4.5`, `moonshotai/kimi-k2.5` — by editing the `model`/`smallFastModel` in `~/.config/cclauncher/config.yaml`.

### wafer

```bash
export WAFER_API_KEY="..."
export WAFER_MODEL="your-model-id"
export WAFER_SMALL_MODEL="$WAFER_MODEL"
ccl --provider wafer
# or shorthand
ccl -p wafer
```

| Field | Value |
|-------|-------|
| Endpoint | `https://pass.wafer.ai` |
| Model | User-selected (`WAFER_MODEL`) |
| Small/Fast | User-selected (`WAFER_SMALL_MODEL`) |
| Env var | `WAFER_API_KEY` |

[Wafer Pass](https://docs.wafer.ai/wafer-pass) routes to frontier open models via an Anthropic-compatible endpoint. Available models include `GLM-5.1`, `DeepSeek-V4.1-Flash`, and `Qwen3.5-397B-A17B` (model names are case-insensitive). Override without editing config:

```bash
export WAFER_MODEL="DeepSeek-V4.1-Flash"
export WAFER_SMALL_MODEL="Qwen3.5-397B-A17B"
```

## Anthropic (OAuth)

### claude

```bash
ccl --provider claude
```

| Field | Value |
|-------|-------|
| Endpoint | `https://api.anthropic.com` |
| Auth | `claude` CLI's own OAuth flow — CCL doesn't manage credentials |

No API key needed. If you haven't authenticated, `claude` will prompt you to log in.

## Local

Local providers connect to a server running on your machine. No API key is required by default, and no internet traffic is sent for the LLM call itself.

See [local-models.md](local-models.md) for server setup instructions and model recommendations.

After configuring each provider's model using the variables below:

```bash
ccl --provider llamabarn   # localhost:2276
ccl --provider lmstudio    # localhost:1234
ccl --provider llamacpp    # localhost:8080
ccl --provider omlx        # localhost:8000
```

### llamabarn

| Field | Value |
|-------|-------|
| Endpoint | `http://localhost:2276` (override: `LLAMABARN_BASE_URL`) |
| Model | User-selected (`LLAMABARN_MODEL`) |
| Small/Fast | User-selected (`LLAMABARN_SMALL_MODEL`) |
| Auth | `LLAMABARN_API_KEY` (optional) |

### lmstudio

| Field | Value |
|-------|-------|
| Endpoint | `http://localhost:1234` (override: `LMSTUDIO_BASE_URL`) |
| Model | User-selected (`LMSTUDIO_MODEL`) |
| Small/Fast | User-selected (`LMSTUDIO_SMALL_MODEL`) |
| Auth | Any non-empty token — defaults to `lmstudio` |

> **Tip:** LM Studio validates that the `Authorization` header is present but doesn't check the token value. The default `lmstudio` token works with any loaded model.

### llamacpp

| Field | Value |
|-------|-------|
| Endpoint | `http://localhost:8080` (override: `LLAMACPP_BASE_URL`) |
| Model | User-selected (`LLAMACPP_MODEL`) |
| Small/Fast | User-selected (`LLAMACPP_SMALL_MODEL`) |
| Auth | Any non-empty token — defaults to `llamacpp` |

`llama-server` supports the Anthropic Messages API natively when started with `--port 8080`. CCL connects directly without a translation layer.

### omlx

| Field | Value |
|-------|-------|
| Endpoint | `http://localhost:8000` (override: `OMLX_BASE_URL`) |
| Model | User-selected (`OMLX_MODEL`) |
| Small/Fast | User-selected (`OMLX_SMALL_MODEL`) |
| Auth | `OMLX_API_KEY`, or local placeholder `omlx` to avoid account OAuth |

`omlx` targets a local MLX-backed Anthropic-compatible server on macOS. Override the endpoint and models without editing config:

```bash
export OMLX_BASE_URL="http://localhost:8000"
export OMLX_MODEL="your-loaded-model-id"
export OMLX_SMALL_MODEL="$OMLX_MODEL"
```

## Switching Providers

Select a provider with `--provider`:

```bash
ccl --provider deepseek "review this PR"
```

Run bare `ccl` at an interactive terminal to open the provider picker. It pre-selects the last successfully completed provider/model pair. Type a displayed number and press Enter to select quickly (including multi-digit numbers), or keep using arrow keys and `/` filtering. Backspace corrects a number; out-of-range numbers cannot submit. After a session exits with status 0, CCL saves the selected pair as the next default; failures and metadata/admin commands do not change it.

Defaults are saved under `cli` in `~/.config/cclauncher/config.yaml`, preserving provider templates and credentials. When stdin is not a terminal, bare `ccl` uses that default without prompting. See [configuration.md](configuration.md) for model override precedence.

## Environment Variable Overrides

Every provider supports a `CCL_<PROVIDER>_API_KEY` override that takes precedence over the config file:

```bash
export CCL_SYNTHETIC_API_KEY="sk-other-key"   # overrides SYNTHETIC_API_KEY for synthetic
export CCL_DEEPSEEK_API_KEY="sk-other-key"    # overrides DEEPSEEK_API_KEY for deepseek
```

Full auth priority: `CCL_<PROVIDER>_API_KEY` > config `authToken` (with `${VAR}` interpolation) > built-in default.

See [configuration.md](configuration.md) for the full interpolation reference.

For names containing hyphens, credential overrides normalize them to underscores
(`my-provider` uses `CCL_MY_PROVIDER_API_KEY`). The legacy hyphenated spelling is
accepted only when the normalized variable is absent. Present empty values clear all
configured credentials. Native Claude login remains available without a configured
key for direct launches; request transformations require an upstream key.
