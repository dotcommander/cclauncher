# CCL Docs

```bash
go install github.com/dotcommander/cclauncher/cmd/ccl@latest
export DEEPSEEK_API_KEY="sk-..."
ccl --provider deepseek -p "say hello"
```

Use these docs when you want to run Claude Code through a different model
provider without hand-editing `ANTHROPIC_*` environment variables.

## Start Here

| Job | Read |
|-----|------|
| Install and run CCL | [../README.md](../README.md) |
| Set keys, defaults, optimization, and router values | [configuration.md](configuration.md) |
| Pick a cloud, local, or Anthropic provider | [providers.md](providers.md) |
| Run LM Studio, llama.cpp, LlamaBarn, or omlx | [local-models.md](local-models.md) |
| Add a provider to your config or the shipped template | [contributing/adding-providers.md](contributing/adding-providers.md) |

## Common Tasks

### List providers

```bash
ccl providers
```

The table comes from `~/.config/cclauncher/config.yaml` after environment
variable interpolation and CCL's default-provider merge. It may show different
models than the committed defaults if you already customized your local config.

### Check provider configuration

```bash
ccl doctor
ccl doctor --provider deepseek
ccl doctor --json
ccl doctor --check-net
```

The default checks are local. `--check-net` also probes provider reachability.
Use `--json` for scripts and CI.

### Switch for one run

```bash
ccl --provider deepseek -p "summarize this repo"
```

Only CCL's provider selector is consumed. Other flags pass through to
`claude`.

### Pick or set a default provider

```bash
ccl
```

Bare `ccl` opens the provider picker at an interactive terminal. It pre-selects `cli.defaultProvider`; press Enter to launch it or pick another provider for that launch only. When stdin is not a terminal, bare `ccl` uses that default without prompting.

Set `cli.defaultProvider` by editing `~/.config/cclauncher/config.yaml`.

### Override one API key

```bash
export CCL_DEEPSEEK_API_KEY="sk-test-key"
ccl --provider deepseek
```

`CCL_<PROVIDER>_API_KEY` overrides the provider's configured `authToken`
without editing the YAML file.

## Development

```bash
go build -o ccl ./cmd/ccl
mkdir -p "$(go env GOPATH)/bin"
ln -sf "$(pwd)/ccl" "$(go env GOPATH)/bin/ccl"
go test ./...
ccl --help
```

Rebuild after code changes so the Go bin-directory symlink points at the binary
you just built.
