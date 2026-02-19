# fitz

Small CLI with a stable v1 command surface.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/alexneyler/fitz/main/install.sh | sh
```

The installer downloads the latest release binary into `~/.fitz/bin/fitz`, adds `~/.fitz/bin` to PATH in your shell rc file, appends shell completion setup, and installs the `create-pr` and `update-fitz` Copilot skills into `~/.agents/skills/`.

## Commands

### Human commands

- `fitz br` — manage worktrees.
  - `fitz br` — interactive worktree list with key bindings (↑/↓: navigate, enter: go, d: delete, n: new, p: publish, q: quit).
  - `fitz br new [--base <branch>] <name> [prompt...]` — create a new worktree. Optionally set a base branch with `--base`. If a prompt is given (one or more words), copilot runs in the background with `--yolo`.
  - `fitz br go <name>` — switch to a worktree.
  - `fitz br rm <name> [--force]` — remove a worktree and its branch.
  - `fitz br rm --all [--force]` — remove all worktrees and their branches.
  - `fitz br list` — interactive worktree list (same as `fitz br`). Shows Copilot session activity plus `fitz agent status` updates, including clickable PR links.
  - `fitz br cd <name>` — print the path to a worktree (for shell integration).
  - `fitz br publish [name]` — push the current branch and open a pull request via Copilot CLI (uses the `create-pr` skill). Optionally specify a worktree name.
  - `fitz br help` — show br usage and available subcommands.
- `fitz completion <bash|zsh>` — print completion script for your shell.
- `fitz config [--global] <command>` — get and set configuration values.
  - `fitz config get <key>` — print the value of a config key (repo-level).
  - `fitz config set <key> <value>` — set a config key (repo-level).
  - `fitz config unset <key>` — remove a config key (repo-level).
  - `fitz config list` — list all config keys and their values (repo-level).
  - Add `--global` to any subcommand to target global config (`~/.fitz/config.json`) instead.
  - Valid keys: `model` (passed as `--model` to Copilot CLI), `agent` (agent framework; default: `copilot-cli`).
  - Config is stored at `~/.fitz/<owner>/<repo>/config.json` (repo-level) or `~/.fitz/config.json` (global). Defaults: `model=gpt-5.3-codex`, `agent=copilot-cli`. Repo config overrides global, which overrides defaults.
  - `fitz config help` — show config usage and available subcommands.
- `fitz help` — print usage.
- `fitz todo` — quick per-repo todo list.
  - `fitz todo <text>` — add a new todo item.
  - `fitz todo list` — interactive TUI (enter: create worktree, d: mark done, add new inline).
  - `fitz todo help` — show todo usage and available subcommands.
- `fitz update` — replace the current executable with the latest release asset for your OS/arch.
- `fitz version` — print current version.

### Agent commands (humans can run these too)

Fitz is built for both humans and agents. Agents can call these commands to report progress, and humans can run them directly when helpful.

- `fitz agent` — workflow commands for agents to execute.
  - `fitz agent status [--pr <url>] [message]` — store branch status metadata for `fitz br list` (message is capped to 80 chars).
  - `fitz agent help` — show agent usage and available subcommands.

## Shell integration (bash/zsh)

Installer behavior:
- bash: appends `eval "$(fitz completion bash)"` to `~/.bashrc`.
- zsh: appends `eval "$(fitz completion zsh)"` to `~/.zshrc`.

Manual setup (if needed):
- bash: add `eval "$(fitz completion bash)"` to `~/.bashrc`.
- zsh: add `eval "$(fitz completion zsh)"` to `~/.zshrc`.

## Update + release artifact naming

`fitz update` calls the latest GitHub release API and only accepts an exact asset name format: `fitz_<goos>_<goarch>` (or `fitz_<goos>_<goarch>.exe` on Windows).

## Local development

```sh
# run tests
make test

# run locally
go run ./cmd/fitz help
go run ./cmd/fitz version

# build local binary
make build
./bin/fitz help

# build local release-named artifact (version defaults to "dev")
make release-local

# build with a specific version
make release-local VERSION=v0.1.0
```

## CI / CD

PR checks run `make lint` and `make test` on every pull request to `main`.

Pushing a tag like `v1.0.0` triggers a release workflow that cross-compiles all platform binaries and creates a GitHub Release with auto-generated notes.
