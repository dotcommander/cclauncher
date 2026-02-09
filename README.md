# CCL - Claude Code Launcher

<p align="center"><a href="https://synthetic.new/?referral=jjrgrqtQXNCBf6C">Built with Synthetic.new API</a></p>

A lightweight Go launcher for Claude Code that enables switching between LLM providers without complex configuration. Providers expose an Anthropic-compatible Messages API, so Claude Code works natively with each one.

## Installation

```bash
git clone https://github.com/vampirefrog/cclauncher ~/go/src/cclauncher
cd ~/go/src/cclauncher

# Build and install (requires just)
just install

# Or manually
go build -ldflags='-s -w' -o ccl ./cmd/ccg
ln -sf $(pwd)/ccl ~/go/bin/ccl
```

Ensure `~/go/bin` is in your `PATH`.

## Usage

```bash
ccl                    # Default provider (Synthetic/Kimi-K2.5)
ccl -p deepseek        # DeepSeek-V3.2 via Synthetic.new
ccl -p kimi2           # Kimi-K2.5 via Synthetic.new
ccl -p minimax         # MiniMax-M2 via Synthetic.new
ccl -p qwen            # Qwen3-VL-235B via Synthetic.new
ccl -p qwen3-coder     # Qwen3-Coder-480B via Synthetic.new
ccl -p claude          # Anthropic Claude (OAuth)
ccl -p claude2         # Anthropic Claude (second account)
ccl -p zai             # Z.ai (GLM-4.7)
ccl -p llamabarn       # LlamaBarn (local)
```

All arguments after the provider flag are passed through to `claude`.

## Providers

| Provider | Model | API |
|----------|-------|-----|
| **synthetic** (default) | Kimi-K2.5 | Synthetic.new |
| **kimi2** | Kimi-K2.5 | Synthetic.new |
| **deepseek** | DeepSeek-V3.2 | Synthetic.new |
| **minimax** | MiniMax-M2 | Synthetic.new |
| **qwen** | Qwen3-VL-235B | Synthetic.new |
| **qwen3-coder** | Qwen3-Coder-480B | Synthetic.new |
| **claude** | Claude (default) | Anthropic |
| **claude2** | Claude (OAuth) | Anthropic |
| **zai** | GLM-4.7 | Z.ai |
| **llamabarn** | Local model | LlamaBarn |

## Configuration

API keys via environment variables:

```bash
export SYNTHETIC_API_KEY="..."      # Synthetic.new providers
export ZAI_API_KEY="..."            # Z.ai
export LLAMABARN_API_KEY="..."      # LlamaBarn
export CLAUDE2_OAUTH_TOKEN="..."    # Claude second account
```

Optional config file at `~/.config/cclauncher/config.yaml` with `${VAR:-default}` interpolation. See `examples/config.yaml.example`.

## Architecture

CCL sets the appropriate `ANTHROPIC_*` environment variables for the chosen provider, then launches the `claude` CLI. All providers use the Anthropic Messages API format natively, so no proxy or API translation is needed.

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
