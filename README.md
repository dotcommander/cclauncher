# CCL — Claude Code Launcher

Switch Claude Code between LLM providers without touching a config file. CCL sets the right `ANTHROPIC_*` environment variables and hands off to `claude` — no proxy or translation layer runs unless you configure transformer rules.

## Installation

```bash
go install github.com/dotcommander/cclauncher/cmd/ccl@latest
```

Ensure `~/go/bin` is in your `PATH`. To build from source:

```bash
git clone https://github.com/dotcommander/cclauncher
cd cclauncher
just install
```

## Quick Start

```bash
# Launch with the default provider (Z.ai / GLM-4.7) — requires ZAI_API_KEY
ccl

# Launch with a specific provider
ccl --provider deepseek

# All other flags pass through to claude
ccl --provider synthetic "fix the null pointer in main.go"
ccl --provider deepseek -c -p "/dc:next"
```

## Commands

| Command | Description |
|---------|-------------|
| `ccl` | Launch with the default provider |
| `ccl --provider <name>` / `ccl -p <name>` | Select a provider for this session |
| `ccl use <provider>` | Persist a provider as the new default |
| `ccl providers` | List all configured providers with auth status |
| `ccl update [--check]` | Update CCL to the latest version |
| `ccl version` | Print the installed version |

> **Tip:** `-p` is overloaded. CCL owns it for provider selection — but only as the **first** argument pair. Claude Code also uses `-p` for print mode. Rule: the first `-p`/`--provider` goes to CCL, every flag after it passes through to `claude` verbatim.
>
> ```bash
> ccl -p deepseek -p "fix this bug"   # provider=deepseek, claude runs in print mode
> ccl -p "fix this bug"               # ERROR — "fix this bug" is not a known provider
> ```
>
> If your first argument looks like a provider name, CCL will claim it. Use `--provider` (long form) when in doubt.

## Providers at a Glance

CCL ships with 11 pre-configured providers across three categories:

**Cloud (native APIs)** — `synthetic`, `deepseek`, `minimax`, `zai`, `openrouter`, `wafer`:

```bash
export SYNTHETIC_API_KEY="sk-..."
ccl --provider synthetic     # GLM-4.7 via Synthetic.new

export DEEPSEEK_API_KEY="sk-..."
ccl --provider deepseek

export OPENROUTER_API_KEY="sk-or-..."
ccl --provider openrouter    # Anthropic Skin — any OpenRouter model

export WAFER_API_KEY="..."
ccl --provider wafer         # Wafer Pass — frontier open models
```

**Local** — `llamabarn`, `lmstudio`, `llamacpp`, `omlx` — no API key required, model server must be running locally.

**Anthropic (OAuth)** — `claude` — authentication handled by the `claude` CLI itself.

See [docs/providers.md](docs/providers.md) for the full reference with endpoints, env vars, and per-provider quirks.

## Configuration

CCL creates `~/.config/cclauncher/config.yaml` on first run with all providers pre-configured. You only need to set the relevant API key:

```bash
export SYNTHETIC_API_KEY="sk-..."    # synthetic
export DEEPSEEK_API_KEY="sk-..."     # deepseek
export MINIMAX_API_KEY="sk-..."      # minimax
export ZAI_API_KEY="sk-..."          # zai
export OPENROUTER_API_KEY="sk-or-..." # openrouter
```

To persist a default provider:

```bash
ccl use deepseek
```

See [docs/configuration.md](docs/configuration.md) for the full config schema, env var interpolation rules, and optimization settings.

## Local Models

```bash
ccl --provider lmstudio    # LM Studio on localhost:1234
ccl --provider llamacpp    # llama.cpp server on localhost:8080
ccl --provider llamabarn   # LlamaBarn on localhost:2276
```

See [docs/local-models.md](docs/local-models.md) for server setup and model recommendations.

## Development

```bash
just build    # Build ./ccl
just install  # Build + symlink to ~/go/bin/ccl
just test     # Run tests
just lint     # golangci-lint
just fmt      # gofmt
just dev      # go run (no build)
just clean    # Remove artifacts
```

## License

MIT
