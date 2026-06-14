# Local Models

CCL supports four local inference servers: LlamaBarn, LM Studio, llama.cpp, and omlx. All expose an API that Claude Code can reach directly — no internet connection needed for inference.

> **Tip:** Claude Code is tool-heavy. It sends many small requests and expects fast responses. For agentic tasks, use a model of at least 30B parameters with strong instruction-following: Qwen3-Coder, Kimi-K2, MiniMax-M2, or a similarly capable coder/instruct model.

## LM Studio

LM Studio runs a local OpenAI-compatible server. CCL uses it via the `lmstudio` provider.

### Start the server

Open LM Studio, load a model, and start the local server on port 1234. Or use the CLI:

```bash
lms server start
```

### Launch Claude Code

```bash
ccl --provider lmstudio
```

By default CCL connects to `http://localhost:1234` with model `openai/gpt-oss-20b`. To use a different model:

```bash
export LMSTUDIO_MODEL="lmstudio-community/Qwen3-Coder-30B-A3B-GGUF"
ccl --provider lmstudio
```

> **Tip:** LM Studio requires a non-empty `Authorization` header but doesn't validate the token value. CCL defaults to `lmstudio` — this works for any loaded model.

### Environment variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `LMSTUDIO_BASE_URL` | `http://localhost:1234` | Server address |
| `LMSTUDIO_MODEL` | `openai/gpt-oss-20b` | Model ID (must match what LM Studio loaded) |
| `LMSTUDIO_SMALL_MODEL` | `openai/gpt-oss-20b` | Background/subagent model |
| `LMSTUDIO_API_KEY` | `lmstudio` | Auth token (any non-empty value works) |

### Config override

To make these settings permanent, edit `~/.config/cclauncher/config.yaml`:

```yaml
providers:
  lmstudio:
    baseUrl: "http://localhost:1234"
    authToken: "lmstudio"
    model: "lmstudio-community/Qwen3-Coder-30B-A3B-GGUF"
    smallFastModel: "lmstudio-community/Qwen3-Coder-30B-A3B-GGUF"
```

## llama.cpp

`llama-server` includes native Anthropic Messages API support. CCL connects to it directly.

### Start the server

```bash
llama-server \
  --model /path/to/model.gguf \
  --port 8080 \
  --ctx-size 32768 \
  --n-gpu-layers 99
```

For agentic use, enable the Anthropic-compatible endpoint explicitly if your build requires it:

```bash
llama-server --model /path/to/model.gguf --port 8080
```

### Launch Claude Code

```bash
ccl --provider llamacpp
```

### Environment variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `LLAMACPP_BASE_URL` | `http://localhost:8080` | Server address |
| `LLAMACPP_MODEL` | `local` | Model identifier sent in requests |
| `LLAMACPP_SMALL_MODEL` | `local` | Background/subagent model identifier |
| `LLAMACPP_API_KEY` | `llamacpp` | Auth token (any non-empty value works) |

> **Note:** `llama-server` ignores the model field in requests — it serves whatever model it loaded at startup. The `LLAMACPP_MODEL` value is passed through but has no effect on which weights are used.

### Config override

```yaml
providers:
  llamacpp:
    baseUrl: "http://localhost:8080"
    authToken: "llamacpp"
    model: "local"
    smallFastModel: "local"
```

## LlamaBarn

LlamaBarn is a local model runner that exposes an OpenAI-compatible API on port 2276.

Key local paths:

| What | Value |
|------|-------|
| App | `/Applications/LlamaBarn.app` |
| API | `http://localhost:2276/v1` |
| llama-server | `/opt/homebrew/bin/llama-server` |
| Upstream | `ggml-org/LlamaBarn` |

Update with:

```bash
brew upgrade --cask llamabarn
brew upgrade llama.cpp
```

### Start the server

Follow the LlamaBarn documentation for your platform. Configure CCL with the server root, `http://localhost:2276`; CCL appends `/v1/messages` when launching Claude Code.

### Launch Claude Code

```bash
ccl --provider llamabarn
```

### Environment variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `LLAMABARN_BASE_URL` | `http://localhost:2276` | Server root URL |
| `LLAMABARN_MODEL` | `local` | Model to request |
| `LLAMABARN_SMALL_MODEL` | `local` | Background/subagent model |
| `LLAMABARN_API_KEY` | *(empty)* | API key if your server requires one |

### Config override

```yaml
providers:
  llamabarn:
    baseUrl: "http://localhost:2276"
    authRequired: false
    authToken: "${LLAMABARN_API_KEY}"
    model: "${LLAMABARN_MODEL:-local}"
    smallFastModel: "${LLAMABARN_SMALL_MODEL:-local}"
```

## omlx

`omlx` targets a local MLX-backed Anthropic-compatible server on macOS (Apple Silicon).

### Launch Claude Code

```bash
ccl --provider omlx
```

By default CCL connects to `http://localhost:8000` with primary model `Qwen3.5-9B-MLX-4bit` and small/fast model `gemma-4-e2b-it-4bit`.

### Environment variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `OMLX_BASE_URL` | `http://localhost:8000` | Server address |
| `OMLX_MODEL` | `Qwen3.5-9B-MLX-4bit` | Primary model |
| `OMLX_SMALL_MODEL` | `gemma-4-e2b-it-4bit` | Background/subagent model |
| `OMLX_API_KEY` | *(empty)* | API key if your server requires one |

### Config override

```yaml
providers:
  omlx:
    baseUrl: "${OMLX_BASE_URL:-http://localhost:8000}"
    authRequired: false
    model: "${OMLX_MODEL:-Qwen3.5-9B-MLX-4bit}"
    smallFastModel: "${OMLX_SMALL_MODEL:-gemma-4-e2b-it-4bit}"
```

## Switching Servers Mid-Session

Claude Code doesn't support mid-session provider swaps — the environment variables are set at launch and don't change. To switch models, exit Claude Code and relaunch:

```bash
# First session: LM Studio
ccl --provider lmstudio

# Next session: llama.cpp with a different model
export LLAMACPP_MODEL="local"
ccl --provider llamacpp
```

To make a local provider your default:

```bash
ccl use lmstudio
```

## Model Recommendations

Claude Code is an agentic tool — it reads files, runs commands, and edits code in tight loops. Smaller models struggle with tool-call formatting and multi-step reasoning. These models work well locally:

| Model | Size | Strengths |
|-------|------|-----------|
| Qwen3-Coder-30B-A3B | 30B MoE | Strong code generation, tool calls |
| Qwen3-Coder-480B-A35B | 480B MoE | Best coding quality (requires large server) |
| Kimi-K2 | 32B | Solid general + code reasoning |
| MiniMax-M2 | varies | Long context, good instruction following |

> **Warning:** Models under 14B parameters tend to produce malformed tool calls, which cause Claude Code to loop or stall. Use at least a 30B model for reliable agentic sessions.

### Why the 30B floor

Claude Code is a tool-calling loop. Read a file → decide → call a tool → read the result → decide again. A single session can chain 50+ tool calls, each one a JSON object the model must emit without a single misplaced brace.

What happens when a 7B model misplaces one brace at call 23? The tool call fails to parse. Claude Code retries. The model makes the same class of mistake. It loops, or it stalls. Your session is dead.

30B+ instruction-tuned models hold tool-call grammar cleanly across long traces. 14B-and-below models do not — not because they can't code, but because they can't stay grammatically perfect for 50 consecutive structured outputs.

Fit isn't the constraint — grammatical stamina is.
