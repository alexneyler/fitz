# fitz

Small CLI with a stable v1 command surface.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/alexneyler/fitz/main/install.sh | sh
```

The installer downloads the latest release binary into `~/.fitz/bin/fitz`, adds `~/.fitz/bin` to PATH in your shell rc file, and appends shell completion setup.

## Commands

- `fitz br` — manage worktrees.
  - `fitz br` — show current worktree.
  - `fitz br new [--base <branch>] <name> [prompt...]` — create a new worktree. Optionally set a base branch with `--base`. If a prompt is given (one or more words), copilot runs in the background with `--yolo`.
  - `fitz br go <name>` — switch to a worktree.
  - `fitz br rm <name> [--force]` — remove a worktree.
  - `fitz br list` — list all worktrees (current highlighted with `*`).
  - `fitz br cd <name>` — print the path to a worktree (for shell integration).
  - `fitz br help` — show br usage and available subcommands.
- `fitz completion <bash|zsh>` — print completion script for your shell.
- `fitz help` — print usage.
- `fitz todo` — quick per-repo todo list.
  - `fitz todo <text>` — add a new todo item.
  - `fitz todo list` — interactive TUI to view and mark todos done.
  - `fitz todo help` — show todo usage and available subcommands.
- `fitz update` — replace the current executable with the latest release asset for your OS/arch.
- `fitz version` — print current version.

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
