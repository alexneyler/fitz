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
  - `fitz br new <name> [base]` — create a new worktree.
  - `fitz br go <name>` — switch to a worktree.
  - `fitz br rm <name> [--force]` — remove a worktree.
  - `fitz br list` — list all worktrees.
  - `fitz br cd <name>` — print the path to a worktree (for shell integration).
- `fitz completion <bash|zsh>` — print completion script for your shell.
- `fitz help` — print usage.
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

# build local release-named artifact
make release-local
```

For local version testing, inject a version at build time:

```sh
go run -ldflags "-X main.version=v0.0.0-local" ./cmd/fitz version
```
