# CCL — Claude Code Launcher

CCL runs Claude Code against a different model provider — DeepSeek, GLM, a local model, whatever — without you editing config or exporting environment variables by hand.

It works because Claude Code only speaks the Anthropic Messages API. Any provider that also speaks that API will work with Claude Code; the model just needs to be reachable at the right URL with the right key. CCL knows those URLs and keys, sets them, and launches `claude`.

## How it works

`ccl --provider deepseek` does three things:

1. Reads the `deepseek` entry from your config — base URL, API key, model name.
2. Sets the `ANTHROPIC_*` environment variables to match, and removes any pre-existing `ANTHROPIC_*` / `CLAUDE_CODE_*` variables from the environment so they can't leak through.
3. Replaces itself with `claude` via `syscall.Exec`.

There's no proxy or translation layer in the request path — CCL exits the moment `claude` starts, and the process you're left talking to *is* Claude Code, configured. (Transformer rules for non-Anthropic-shaped APIs exist but are opt-in; nothing runs between you and the model unless you set them up.)

## Install

```bash
go install github.com/dotcommander/cclauncher/cmd/ccl@latest
```

`~/go/bin` must be on your `PATH`. From source:

```bash
git clone https://github.com/dotcommander/cclauncher
cd cclauncher
just install
```

## Usage

```bash
ccl                                  # pick a provider; pre-selects the default
ccl --provider deepseek              # this session only; skips the picker
ccl --provider synthetic "fix the null pointer in main.go"
```

Arguments after the provider flag pass through to `claude` unchanged.

| Command | Description |
|---------|-------------|
| `ccl --provider <name>` | Select a provider for this session |
| `ccl providers` | List providers and whether each key is set |
| `ccl update [--check]` | Update CCL |
| `ccl version` | Print the version |

**Note on `-p`:** CCL uses `-p` for `--provider`; Claude Code uses `-p` for print mode. The first `-p`/`--provider` goes to CCL, everything after passes through to `claude`.

```bash
ccl -p deepseek -p "fix this bug"   # provider=deepseek, claude in print mode
ccl -p "fix this bug"               # error: "fix this bug" is not a provider
```

Use the long form `--provider` to avoid ambiguity.

For the full docs index, see [docs/README.md](docs/README.md).

## Providers

CCL ships with eleven providers in three categories. They're pre-configured; you only supply the key.

**Cloud** — `synthetic`, `deepseek`, `minimax`, `zai`, `openrouter`, `wafer`. Each reads one environment variable:

```bash
export DEEPSEEK_API_KEY="sk-..."
ccl --provider deepseek

export OPENROUTER_API_KEY="sk-or-..."
ccl --provider openrouter
```

**Local** — `llamabarn`, `lmstudio`, `llamacpp`, `omlx`. No key; the model server runs on your machine and CCL points at `localhost`:

```bash
ccl --provider lmstudio          # LM Studio, localhost:1234
ccl --provider llamacpp          # llama.cpp, localhost:8080
ccl --provider llamabarn         # LlamaBarn, localhost:2276
```

**Anthropic** — `claude`. Auth is handled by the `claude` CLI's own OAuth; nothing to set.

Endpoints, environment variables, and per-provider notes: [docs/providers.md](docs/providers.md).

## Configuration

On first run, CCL writes `~/.config/cclauncher/config.yaml` with all providers filled in. You supply the API key through an environment variable; CCL can't guess it.

Run bare `ccl` at an interactive terminal to open the provider picker. It pre-selects the configured default, so press Enter to launch it or pick another provider for that launch only. In pipes or CI, bare `ccl` uses the configured default without prompting.

Set that default by editing `cli.defaultProvider` in `~/.config/cclauncher/config.yaml`.

Override a single key without editing the config — `CCL_<PROVIDER>_API_KEY` takes precedence:

```bash
export CCL_DEEPSEEK_API_KEY="sk-..."
```

If you select a provider whose key is missing, CCL stops before launching and tells you which variable to set, rather than producing a 401 mid-session.

Config schema and interpolation rules: [docs/configuration.md](docs/configuration.md). Local model setup: [docs/local-models.md](docs/local-models.md).

## Development

```bash
just build      # build ./ccl
just install    # build + symlink to ~/go/bin/ccl
just test       # run tests
just lint       # golangci-lint
just dev        # go run, no build
```

Providers are defined in YAML, not Go. To add one, update `internal/config/default-config.yaml`, then run `go run ./internal/tools/gen-config-examples` so committed examples stay generated from the canonical default config. CCL sets provider env vars generically. See [docs/providers.md](docs/providers.md).

## License

MIT
