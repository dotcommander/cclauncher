# Providers

CCL ships with 14 pre-configured providers across four categories. Each provider maps to a specific model and API endpoint. CCL sets the correct `ANTHROPIC_*` environment variables and calls `claude` — no proxy or translation layer runs unless you configure transformer rules.

## Aggregator (Synthetic.new)

These providers all share one API key (`SYNTHETIC_API_KEY`) and route through `https://api.synthetic.new/anthropic`.

```bash
export SYNTHETIC_API_KEY="sk-..."
ccl                          # synthetic — Kimi-K2.5 (default)
ccl --provider kimi2         # kimi2 — Kimi-K2.5 (explicit)
ccl --provider qwen          # qwen — Qwen3-235B-A22B-Thinking
ccl --provider qwen3-coder   # qwen3-coder — Qwen3-Coder-480B
ccl --provider deepseek-synthetic   # DeepSeek-V3.2 via Synthetic.new
ccl --provider minimax-synthetic    # MiniMax-M2.1 via Synthetic.new
```

| Provider | Model | Small/Fast Model |
|----------|-------|-----------------|
| `synthetic` | `hf:moonshotai/Kimi-K2.5` | `hf:moonshotai/Kimi-K2.5` |
| `kimi2` | `hf:moonshotai/Kimi-K2.5` | `hf:moonshotai/Kimi-K2.5` |
| `qwen` | `hf:Qwen/Qwen3-235B-A22B-Thinking-2507` | same |
| `qwen3-coder` | `hf:Qwen/Qwen3-Coder-480B-A35B-Instruct` | same |
| `deepseek-synthetic` | `hf:deepseek-ai/DeepSeek-V3.2` | same |
| `minimax-synthetic` | `hf:MiniMaxAI/MiniMax-M2.1` | same |

`synthetic` and `kimi2` use the same model — `synthetic` is the default provider, `kimi2` is an explicit alias. Use `kimi2` in scripts where model intent should be obvious from the command; use `synthetic` in interactive sessions where "the default" is the clearer read.

## Cloud (Native APIs)

These providers use their own API endpoints and require separate keys.

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

## Anthropic (OAuth)

These providers connect to `https://api.anthropic.com`. Authentication is handled by the `claude` CLI's own OAuth flow — CCL doesn't manage credentials for them.

### claude

```bash
ccl --provider claude
```

No API key needed. If you haven't authenticated, `claude` will prompt you to log in.

### claude2

```bash
export CLAUDE2_OAUTH_TOKEN="..."
ccl --provider claude2
```

`claude2` uses an explicit OAuth token stored in `CLAUDE2_OAUTH_TOKEN`. This lets you run a second Anthropic account alongside your primary one.

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
ccl --provider minimax "review this PR"
```

Persist a new default with `use`:

```bash
ccl use deepseek
```

`ccl use` writes `cli.defaultProvider` to `~/.config/cclauncher/config.yaml` while preserving all comments and existing values.

## Environment Variable Overrides

Every provider supports a `CCL_<PROVIDER>_API_KEY` override that takes precedence over the config file:

```bash
export CCL_SYNTHETIC_API_KEY="sk-other-key"   # overrides SYNTHETIC_API_KEY for synthetic/kimi2/qwen/...
export CCL_DEEPSEEK_API_KEY="sk-other-key"    # overrides DEEPSEEK_API_KEY for deepseek
```

Full auth priority: `CCL_<PROVIDER>_API_KEY` > config `authToken` (with `${VAR}` interpolation) > built-in default.

See [configuration.md](configuration.md) for the full interpolation reference.
