# Providers

CCL ships with 9 pre-configured providers across three categories. Each provider maps to a specific model and API endpoint. CCL sets the correct `ANTHROPIC_*` environment variables and calls `claude` — no proxy or translation layer runs unless you configure transformer rules.

## Cloud (Native APIs)

These providers use their own API endpoints and require separate keys.

### synthetic

```bash
export SYNTHETIC_API_KEY="sk-..."
ccl --provider synthetic
```

| Field | Value |
|-------|-------|
| Endpoint | `https://api.synthetic.new/anthropic` |
| Model | `hf:zai-org/GLM-4.7` |
| Small/Fast | `hf:zai-org/GLM-4.7` |
| Env var | `SYNTHETIC_API_KEY` |

`synthetic` routes through Synthetic.new to access GLM-4.7. Swap the model for any HuggingFace slug Synthetic hosts by editing `~/.config/cclauncher/config.yaml`.

### deepseek

```bash
export DEEPSEEK_API_KEY="sk-..."
ccl --provider deepseek
```

| Field | Value |
|-------|-------|
| Endpoint | `https://api.deepseek.com/anthropic` |
| Model | `deepseek-chat` |
| Small/Fast | `deepseek-chat` |
| Env var | `DEEPSEEK_API_KEY` |

### minimax

```bash
export MINIMAX_API_KEY="sk-..."
ccl --provider minimax
```

| Field | Value |
|-------|-------|
| Endpoint | `https://api.minimax.io/anthropic` |
| Model | `MiniMax-M2.7` |
| Small/Fast | `MiniMax-M2.7-highspeed` |
| Env var | `MINIMAX_API_KEY` |

### zai

```bash
export ZAI_API_KEY="sk-..."
ccl --provider zai
```

| Field | Value |
|-------|-------|
| Endpoint | `https://api.z.ai/api/anthropic` |
| Model | `glm-4.7` (override: `ZAI_MODEL`) |
| Small/Fast | `glm-4.5-air` (override: `ZAI_SMALL_MODEL`) |
| Env var | `ZAI_API_KEY` |

`zai` supports model overrides without editing the config file:

```bash
export ZAI_MODEL="glm-4.7"
export ZAI_SMALL_MODEL="glm-4.5-air"
```

### openrouter

```bash
export OPENROUTER_API_KEY="sk-or-..."
ccl --provider openrouter
```

| Field | Value |
|-------|-------|
| Endpoint | `https://openrouter.ai/api` |
| Model | `deepseek/deepseek-v3.2` |
| Small/Fast | `deepseek/deepseek-v3.2` |
| Env var | `OPENROUTER_API_KEY` |

OpenRouter exposes an Anthropic Messages-compatible route at `/api` (its "Anthropic Skin"). Swap the model to any [OpenRouter slug](https://openrouter.ai/models) — e.g. `anthropic/claude-sonnet-4.5`, `moonshotai/kimi-k2.5` — by editing the `model`/`smallFastModel` in `~/.config/cclauncher/config.yaml`.

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

```bash
ccl --provider llamabarn   # localhost:2276
ccl --provider lmstudio    # localhost:1234
ccl --provider llamacpp    # localhost:8080
```

### llamabarn

| Field | Value |
|-------|-------|
| Endpoint | `http://localhost:2276/v1` (override: `LLAMABARN_BASE_URL`) |
| Model | `local` (override: `LLAMABARN_MODEL`) |
| Auth | `LLAMABARN_API_KEY` (optional) |

### lmstudio

| Field | Value |
|-------|-------|
| Endpoint | `http://localhost:1234` (override: `LMSTUDIO_BASE_URL`) |
| Model | `openai/gpt-oss-20b` (override: `LMSTUDIO_MODEL`) |
| Auth | Any non-empty token — defaults to `lmstudio` |

> **Tip:** LM Studio validates that the `Authorization` header is present but doesn't check the token value. The default `lmstudio` token works with any loaded model.

### llamacpp

| Field | Value |
|-------|-------|
| Endpoint | `http://localhost:8080` (override: `LLAMACPP_BASE_URL`) |
| Model | `local` (override: `LLAMACPP_MODEL`) |
| Auth | Any non-empty token — defaults to `llamacpp` |

`llama-server` supports the Anthropic Messages API natively when started with `--port 8080`. CCL connects directly without a translation layer.

## Switching Providers

Switch for a single session with `--provider`:

```bash
ccl --provider deepseek "review this PR"
```

Persist a new default with `use`:

```bash
ccl use deepseek
```

`ccl use` writes `cli.defaultProvider` to `~/.config/cclauncher/config.yaml` while preserving all comments and existing values.

## Environment Variable Overrides

Every provider supports a `CCL_<PROVIDER>_API_KEY` override that takes precedence over the config file:

```bash
export CCL_SYNTHETIC_API_KEY="sk-other-key"   # overrides SYNTHETIC_API_KEY for synthetic
export CCL_DEEPSEEK_API_KEY="sk-other-key"    # overrides DEEPSEEK_API_KEY for deepseek
```

Full auth priority: `CCL_<PROVIDER>_API_KEY` > config `authToken` (with `${VAR}` interpolation) > built-in default.

See [configuration.md](configuration.md) for the full interpolation reference.
