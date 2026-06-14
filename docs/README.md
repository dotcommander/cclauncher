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

### Switch for one run

```bash
ccl --provider deepseek -p "summarize this repo"
```

Only CCL's provider selector is consumed. Other flags pass through to
`claude`.

### Persist a default provider

```bash
ccl use deepseek
ccl "run the tests"
```

`ccl use` writes `cli.defaultProvider` in
`~/.config/cclauncher/config.yaml`.

### Override one API key

```bash
export CCL_DEEPSEEK_API_KEY="sk-test-key"
ccl --provider deepseek
```

`CCL_<PROVIDER>_API_KEY` overrides the provider's configured `authToken`
without editing the YAML file.

## Development

```bash
just install
go test ./...
ccl --help
```

Use `just install` after code changes so the `~/go/bin/ccl` symlink points at
the binary you just built.
