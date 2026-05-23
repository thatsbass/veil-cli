# veil-cli

Veil CLI is the command-line client for the [Veil](https://veil.dev) LLM
gateway. It authenticates your machine, routes API calls from local AI tools
through a single endpoint, and provides a terminal dashboard for monitoring
usage and logs.

## Quick start

```sh
# Install
git clone https://github.com/thatsbass/veil-cli.git
cd veil-cli
make install

# Authenticate (opens your browser)
veil auth login

# Route your tools through Veil
veil use claude
veil use cursor

# Check status
veil up
```

## Commands

| Command | Description |
|---------|-------------|
| `veil auth login` | Authenticate via device authorization flow |
| `veil auth logout` | Remove the local API key |
| `veil auth status` | Show authentication state |
| `veil use [tool]` | Configure a local tool to route through Veil |
| `veil up` | Display server status and monthly token usage |
| `veil status` | Display local configuration |
| `veil stats` | Show monthly usage and savings |
| `veil logs` | Stream live server logs (Ctrl+C to stop) |
| `veil doctor` | Run a connectivity diagnostic |
| `veil down` | Delete the local session |
| `veil version` | Print the current version |
| `veil update` | Check for available updates |

### Interactive REPL

Running `veil` with no arguments launches an interactive terminal interface with
slash commands:

```
/status     Server status
/stats      Monthly usage
/billing    Current plan and quota
/logs       Live log stream (Esc to stop)
/config     Edit api_url
/use        Show configured tools
/doctor     Connectivity diagnostic
/login      Print login instructions
/logout     Remove API key
/help       Show all commands
/exit       Quit
```

## Supported tools

| Tool | Config file | Keys written |
|------|-------------|--------------|
| Claude Code | `~/.claude/settings.json` | `apiBaseUrl`, `apiKey` |
| Cursor | `~/.cursor/settings.json` | `openai.apiBase`, `openai.apiKey` (nested) |
| Codex CLI | `~/.codex/config.toml` | `api_base_url`, `api_key` |
| Aider | `~/.aider.conf.yml` | `openai-api-base`, `openai-api-key` |

Each `veil use` invocation creates a backup at `<config>.veil.bak` before writing.

## Development

### Prerequisites

- Go 1.24 or later
- `golangci-lint` (optional, for `make lint`)

### Build

```sh
make build        # compile to bin/veil
make dev          # run without compiling
make test         # run all tests with coverage
make lint         # run golangci-lint
make tidy         # clean go.mod
```

### Architecture

The codebase follows Clean Architecture / Ports and Adapters:

```
cmd/main.go                  Composition root (dependency injection)
internal/
  domain/                    Pure business types (no external imports)
  ports/                     Interfaces the usecase layer depends on
  usecase/                   Application orchestration (testable with mocks)
  adapter/
    api/                     HTTP client for the Veil API
    config/                  JSON configuration repository
    configurator/            Tool-specific config writers (Claude, Cursor, etc.)
  delivery/
    cli/commands/            Cobra command implementations
    repl/                    BubbleTea interactive terminal
```

Dependencies flow inward: `delivery -> adapter -> ports <- usecase`. The `domain`
package has zero external dependencies and serves as the single source of truth
for all value types.

### Testing

```sh
make test
```

Usecase tests use mock implementations of `ports.GatewayClient` and require no
network access. Configurator tests write to temporary directories.

## Configuration

The CLI stores state in `~/.veil/config.json`:

```json
{
  "api_key": "vl_live_...",
  "api_url": "https://api.veil.dev"
}
```

## License

MIT
