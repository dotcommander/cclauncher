# CCL — Claude Code Launcher

CCL runs Claude Code against a different model provider — DeepSeek, GLM, a local model, whatever — without you managing Claude Code's `ANTHROPIC_*` variables by hand. You choose the model in CCL config or provider-specific environment variables.

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
go build -o ccl ./cmd/ccl
mkdir -p "$(go env GOPATH)/bin"
ln -sf "$(pwd)/ccl" "$(go env GOPATH)/bin/ccl"
```

## Usage

First configure the provider's key (if required) and explicit model as shown below.
Shipped defaults do not choose models. Native `claude` uses Claude Code's selection.

```bash
ccl                                  # pick a provider; pre-selects the default
ccl --provider deepseek              # this session only; skips the picker
ccl --provider synthetic "fix the null pointer in main.go"
```

Launch arguments other than CCL provider selectors pass through to `claude` unchanged. The first `--` is consumed by CCL and ends provider parsing; subsequent arguments, including later `--` delimiters, pass through.

| Command | Description |
|---------|-------------|
| `ccl --provider <name>` | Select a provider for this session |
| `ccl providers` | List providers and whether each key is set |
| `ccl doctor [--provider <name>] [--json] [--check-net]` | Check provider configuration and optional reachability |
| `ccl update [--check]` | Update CCL |
| `ccl version` | Print the version |

**Note on `-p`:** CCL uses `-p` for `--provider`; Claude Code uses `-p` for print mode. Only a leading `-p PROVIDER` pair belongs to CCL. Later `-p` arguments belong to Claude Code. CCL recognizes `--provider` before its first `--` delimiter.

```bash
ccl -p deepseek -p "fix this bug"   # provider=deepseek, claude in print mode
ccl -p "fix this bug"               # error: "fix this bug" is not a provider
```

Use the long form `--provider` to avoid ambiguity.

For the full docs index, see [docs/README.md](docs/README.md).

## Providers

CCL ships with eleven provider endpoint templates in three categories. You supply
credentials when required and select models; there are no shipped model fallbacks.

**Cloud** — `synthetic`, `deepseek`, `minimax`, `zai`, `openrouter`, `wafer`.
Set the provider's API key and model variables:

```bash
export DEEPSEEK_API_KEY="sk-..."
export DEEPSEEK_MODEL="your-model-id"
export DEEPSEEK_SMALL_MODEL="$DEEPSEEK_MODEL"
ccl --provider deepseek

export OPENROUTER_API_KEY="sk-or-..."
export OPENROUTER_MODEL="your-model-id"
export OPENROUTER_SMALL_MODEL="$OPENROUTER_MODEL"
ccl --provider openrouter
```

**Local** — `llamabarn`, `lmstudio`, `llamacpp`, `omlx`. No key is required by
default; the server runs on your machine and CCL points at `localhost`. Set each
provider's `*_MODEL` to an ID accepted by its server before launching:

```bash
ccl --provider lmstudio          # LM Studio, localhost:1234
ccl --provider llamacpp          # llama.cpp, localhost:8080
ccl --provider llamabarn         # LlamaBarn, localhost:2276
```

**Anthropic** — `claude`. Auth is handled by the `claude` CLI's own OAuth; nothing to set.

Endpoints, environment variables, and per-provider notes: [docs/providers.md](docs/providers.md).

## Configuration

On first run, CCL writes `~/.config/cclauncher/config.yaml` with provider endpoints
and environment templates. Set `<PROVIDER>_MODEL` or `model` explicitly for any
non-Anthropic provider. Set `<PROVIDER>_SMALL_MODEL` or `smallFastModel` to choose
background/subagent models too. Updating CCL preserves existing config choices;
environment variables apply where your config uses their templates.

Run bare `ccl` at an interactive terminal to open the provider picker. It shows only
launchable providers and pre-selects the configured default if eligible; press Enter
to launch the selection or choose another provider for that launch only. Both input and output must be terminals for the picker. With redirected input or output, bare `ccl` uses the configured default without prompting. Launches with arguments also use that default unless a provider is selected.

Set that default by editing `cli.defaultProvider` in `~/.config/cclauncher/config.yaml`.

Override a single key without editing the config — `CCL_<PROVIDER>_API_KEY` takes precedence:

```bash
export CCL_DEEPSEEK_API_KEY="sk-..."
```

Missing required models or credentials block explicit launches before Claude Code
starts. Such providers are also hidden from the interactive picker.

Config schema and interpolation rules: [docs/configuration.md](docs/configuration.md). Local model setup: [docs/local-models.md](docs/local-models.md).

## Check your setup

```bash
ccl doctor                         # check every configured provider
ccl doctor --provider deepseek     # check one provider
ccl doctor --json                  # emit machine-readable results
ccl doctor --check-net             # also probe provider reachability
```

`ccl doctor` checks credentials, required fields, and base URLs without making
network requests. Add `--check-net` when you also want reachability probes.
Missing optional credentials are warnings; invalid required configuration exits
non-zero. Network probes run concurrently with `doctor.probeConcurrency` from YAML
(default 4; set 1 for sequential probes), preserve sorted provider order, and stop
on caller cancellation. Each probe is limited to five seconds.

Standalone `ccl --help`, `ccl help`, and local command help do not load configuration.
Use `ccl -- --help` or `ccl --provider deepseek --help` for Claude Code help.

`ccl update --check` requires no Go compiler. Release lookup has a ten-second
limit; installation retains caller cancellation without that lookup deadline.
Only a newer stable numeric version is installed. Development, malformed, and
prerelease versions are indeterminate and skip installation.

## Development

```bash
go build ./...
go test ./...
go vet ./...
go run ./cmd/ccl --help
```

Providers are defined in YAML, not Go. To add one, update `internal/config/default-config.yaml`, then run `go run ./internal/tools/gen-config-examples` so committed examples stay generated from the canonical default config. CCL sets provider env vars generically. See [docs/providers.md](docs/providers.md).

## License

MIT
