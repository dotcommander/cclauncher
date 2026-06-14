# CCL — Claude Code Launcher

Claude Code doesn't care who's on the other end of the wire. It speaks the Anthropic Messages API and nothing more. So if some *other* model — DeepSeek, GLM, something running on your own laptop — also speaks that dialect, Claude Code will talk to it and never know the difference.

The only thing standing between you and "Claude Code, but powered by DeepSeek" is a few environment variables pointing at a different address.

You *could* set those by hand every time:

```bash
export ANTHROPIC_BASE_URL="https://api.deepseek.com/anthropic"
export ANTHROPIC_AUTH_TOKEN="sk-..."
export ANTHROPIC_MODEL="deepseek-v4-pro"
# ...and a few more, every single time, getting one wrong eventually
claude
```

That's tedious, and tedious is where bugs live. So CCL does it for you:

```bash
ccl --provider deepseek
```

That's the whole idea. Now let me show you what's actually happening underneath, because it's prettier than you'd expect.

## What CCL really does

When you run `ccl --provider deepseek`, it does three things and then *gets out of the way*:

1. Looks up `deepseek` in a config file and grabs the base URL, your API key, and the model name.
2. Sets the `ANTHROPIC_*` environment variables to those values — and scrubs any stale `ANTHROPIC_*` / `CLAUDE_CODE_*` vars hanging around in your shell, so nothing leaks through to confuse things.
3. Calls `syscall.Exec` to replace itself with `claude`.

That third step is the good part. CCL doesn't *wrap* Claude Code. It doesn't sit in the middle proxying your traffic, translating requests, slowing things down. It hands the steering wheel to `claude` and vanishes — the running process *becomes* Claude Code, with the dials already set. There's no daemon, no proxy, no translation layer. Just an environment, prepared, and then a clean handoff.

(If you do want a translation layer for an API that *isn't* Anthropic-shaped, CCL can wire up transformer rules — but you have to ask for it. By default, nothing runs between you and the model.)

## Installation

```bash
go install github.com/dotcommander/cclauncher/cmd/ccl@latest
```

Make sure `~/go/bin` is on your `PATH`. Or build from source:

```bash
git clone https://github.com/dotcommander/cclauncher
cd cclauncher
just install
```

## Using it

```bash
ccl                                  # default provider (Z.ai / GLM)
ccl --provider deepseek              # pick one for this session
ccl --provider synthetic "fix the null pointer in main.go"
```

Anything you put *after* the provider flag is passed straight through to `claude`, untouched. CCL reads the first flag, then steps aside.

A handful of small commands round it out:

| Command | What it does |
|---------|--------------|
| `ccl use deepseek` | Make a provider your permanent default |
| `ccl providers` | List every provider and whether its key is set |
| `ccl update` | Pull the latest CCL |
| `ccl version` | Print the version |

### One sharp edge worth knowing

The flag `-p` is doing double duty, and that occasionally bites people. CCL uses `-p` for *provider*. Claude Code uses `-p` for *print mode*. CCL's rule: the **first** `-p`/`--provider` belongs to CCL; everything after it belongs to `claude`.

```bash
ccl -p deepseek -p "fix this bug"   # provider = deepseek, claude runs print mode. Works.
ccl -p "fix this bug"               # ERROR — "fix this bug" isn't a provider name
```

When in doubt, spell out `--provider`. No ambiguity, no surprises.

## The providers

CCL ships knowing about eleven providers in three flavors. You don't configure them — they're already there. You just supply the key.

**Cloud models** — `synthetic`, `deepseek`, `minimax`, `zai`, `openrouter`, `wafer`. Each wants one environment variable:

```bash
export DEEPSEEK_API_KEY="sk-..."
ccl --provider deepseek

export OPENROUTER_API_KEY="sk-or-..."
ccl --provider openrouter        # gives you any OpenRouter model in an Anthropic skin
```

**Local models** — `llamabarn`, `lmstudio`, `llamacpp`, `omlx`. No key, no cloud, no bill. You run the model server on your own machine and point CCL at `localhost`:

```bash
ccl --provider lmstudio          # LM Studio, localhost:1234
ccl --provider llamacpp          # llama.cpp server, localhost:8080
ccl --provider llamabarn         # LlamaBarn, localhost:2276
```

**Real Anthropic** — `claude`. The genuine article. Auth is handled by the `claude` CLI's own OAuth, so there's nothing for you to set.

Full details — endpoints, env vars, the per-provider quirks — live in [docs/providers.md](docs/providers.md).

## Configuration

On its first run, CCL writes `~/.config/cclauncher/config.yaml` with every provider already filled in. Open it if you're curious; you mostly won't need to. The one thing it can't guess is your API key, so you hand that over through an environment variable.

If you'd rather not type `--provider` every time:

```bash
ccl use deepseek
```

Now plain `ccl` launches DeepSeek until you change your mind.

Want to override a single key without touching anything? There's an env var for that, shaped `CCL_<PROVIDER>_API_KEY`:

```bash
export CCL_DEEPSEEK_API_KEY="sk-..."   # wins over whatever's in the config file
```

The config schema and interpolation rules are in [docs/configuration.md](docs/configuration.md); local model setup is in [docs/local-models.md](docs/local-models.md).

## One nicety: it checks before it leaps

If you pick a provider but forgot to set its key, CCL stops you *before* launching, with a message that tells you exactly which variable to set — instead of letting you discover the problem as a cryptic 401 three messages into a conversation. Small thing. Saves real annoyance.

## Hacking on CCL

```bash
just build      # build ./ccl
just install    # build + symlink to ~/go/bin/ccl
just test       # run the tests
just lint       # golangci-lint
just dev        # go run, no build
```

Adding a new provider takes zero Go code — providers are just YAML. Add a block to `internal/config/default-config.yaml`, and CCL sets the right env vars for it generically. The recipe is in [CLAUDE.md](CLAUDE.md).

## License

MIT
