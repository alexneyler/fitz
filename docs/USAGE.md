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
- `fitz br new <name> [prompt]` — create a new worktree. If a prompt is given, copilot launches in the background with `--yolo -p "<prompt>"`.
  - Example: `fitz br new feature-login`
  - Example: `fitz br new feature-login "implement user authentication"`
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

## Help output

`fitz` and `fitz help` currently print:

```text
fitz <version>
Usage: fitz <command>

Commands:
  br            Manage worktrees
  completion    Print shell completion script
  help          Show this help message
  update        Update fitz to the latest release
  version       Print version information
```
