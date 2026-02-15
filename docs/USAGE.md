# Usage

## Commands

- `fitz` (no args) — prints help/usage.
  - Example: `fitz`
- `fitz help` — prints help/usage.
  - Example: `fitz help`
- `fitz version` — prints the current CLI version (for example `fitz dev`).
  - Example: `fitz version`
- `fitz update` — downloads the latest release asset for your OS/arch and replaces the current executable.
  - Example: `fitz update`
- `fitz completion bash` — prints bash completion script.
  - Example: `fitz completion bash`
- `fitz completion zsh` — prints zsh completion script.
  - Example: `fitz completion zsh`
- `fitz br` — show the current worktree.
  - Example: `fitz br`
- `fitz br new <name> [base]` — create a new worktree from an optional base branch.
  - Example: `fitz br new feature-login`
  - Example: `fitz br new feature-auth main`
- `fitz br go <name>` — switch to an existing worktree.
  - Example: `fitz br go feature-login`
- `fitz br rm <name> [--force]` — remove a worktree (optionally force removal).
  - Example: `fitz br rm feature-login`
  - Example: `fitz br rm feature-login --force`
- `fitz br list` — list all worktrees.
  - Example: `fitz br list`
- `fitz br cd <name>` — print the path to a worktree (for shell integration).
  - Example: `fitz br cd feature-login`
- `fitz br help` — show br usage and available subcommands.
  - Example: `fitz br help`
- `fitz todo <text>` — add a new todo item for the current repo.
  - Example: `fitz todo "fix the login bug"`
  - Example: `fitz todo remember to update docs`
- `fitz todo list` — interactive TUI to view todos and mark them done.
  - Example: `fitz todo list`
- `fitz todo help` — show todo usage and available subcommands.
  - Example: `fitz todo help`

## Help output

`fitz` and `fitz help` currently print:

```text
fitz <version>
Usage: fitz <command>

Commands:
  br            Manage worktrees
  completion    Print shell completion script
  help          Show this help message
  todo          Quick per-repo todo list
  update        Update fitz to the latest release
  version       Print version information
```
