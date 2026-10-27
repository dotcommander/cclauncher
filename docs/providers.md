# Providers

CCL ships with 15 provider endpoint templates across three categories. Models are user-selected, not shipped defaults. CCL sets the correct `ANTHROPIC_*` environment variables and calls `claude` — no proxy or translation layer runs unless you configure transformer rules.

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
ccl --provider zai
```

| Field | Value |
|-------|-------|
| Endpoint | `https://api.z.ai/api/anthropic` |
| Model | `glm-5.3[1m]` by default; override with `ZAI_MODEL` |
| Small/Fast | `glm-5.3-flash[1m]` by default; override with `ZAI_SMALL_MODEL` |
| Env var | `ZAI_API_KEY` |

`zai` ships GLM 1M-context model defaults, so only the API key is required.
Override models without editing the config file:

```bash
export ZAI_MODEL="your-model-id"
export ZAI_SMALL_MODEL="$ZAI_MODEL"
```

The `[1m]` suffix selects the 1M-token context variant. To use the full
window instead of Claude Code auto-compacting at its default limit, raise
the provider's context size:

```yaml
zai:
  maxContextTokens: 1000000
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

### moonshot

```bash
export MOONSHOT_API_KEY="sk-..."
export MOONSHOT_MODEL="your-model-id"
export MOONSHOT_SMALL_MODEL="$MOONSHOT_MODEL"
ccl --provider moonshot
```

| Field | Value |
|-------|-------|
| Endpoint | `https://api.moonshot.ai/anthropic` |
| Model | User-selected (`MOONSHOT_MODEL`) |
| Small/Fast | User-selected (`MOONSHOT_SMALL_MODEL`) |
| Env var | `MOONSHOT_API_KEY` |

Moonshot exposes an Anthropic-compatible endpoint for its Kimi models — the K2.7 Code series is tuned for coding agents. Pick the model ID from Moonshot's API docs.

### qwen

```bash
export QWEN_API_KEY="sk-..."
export QWEN_MODEL="your-model-id"
export QWEN_SMALL_MODEL="$QWEN_MODEL"
ccl --provider qwen
```

| Field | Value |
|-------|-------|
| Endpoint | `https://dashscope.aliyuncs.com/apps/anthropic` (override: `QWEN_BASE_URL`) |
| Model | User-selected (`QWEN_MODEL`) |
| Small/Fast | User-selected (`QWEN_SMALL_MODEL`) |
| Env var | `QWEN_API_KEY` |

Alibaba Model Studio exposes an Anthropic-compatible endpoint for Qwen models (for example the Qwen3-Coder series). The default endpoint is the pay-as-you-go DashScope host; regional workspace endpoints look like `https://{WorkspaceId}.ap-southeast-1.maas.aliyuncs.com/apps/anthropic`, and subscription plans use dedicated hosts (for example `https://coding.dashscope.aliyuncs.com/apps/anthropic` for the Coding Plan) — override `QWEN_BASE_URL` accordingly. The key must match the endpoint's plan type, or the API returns 401 `invalid_api_key`.

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
ccl --provider ollama      # localhost:11434
ccl --provider vllm        # localhost:8000
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

### ollama

| Field | Value |
|-------|-------|
| Endpoint | `http://localhost:11434` (override: `OLLAMA_BASE_URL`) |
| Model | User-selected (`OLLAMA_MODEL`) |
| Small/Fast | User-selected (`OLLAMA_SMALL_MODEL`) |
| Auth | `OLLAMA_API_KEY`, or local placeholder `ollama` |

Ollama 0.14+ serves the Anthropic Messages API natively. After `ollama serve` and pulling a model, set `OLLAMA_MODEL` to the tag reported by `ollama list`.

### vllm

| Field | Value |
|-------|-------|
| Endpoint | `http://localhost:8000` (override: `VLLM_BASE_URL`) |
| Model | User-selected (`VLLM_MODEL`) |
| Small/Fast | User-selected (`VLLM_SMALL_MODEL`) |
| Auth | `VLLM_API_KEY`, or local placeholder `vllm` |

vLLM's built-in Anthropic-protocol entrypoint serves `/v1/messages` (see vLLM's Anthropic endpoint docs for serving flags); set `VLLM_MODEL` to the served model name. Note the default port 8000 matches `omlx` — override one of them when running both.

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
