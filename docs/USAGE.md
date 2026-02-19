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
- `fitz config [--global] <command>` — get and set configuration values. Config is stored at `~/.fitz/<owner>/<repo>/config.json` (repo-level) or `~/.fitz/config.json` (global). Defaults: `model=gpt-5.3-codex`, `agent=copilot-cli`. Repo config overrides global, which overrides built-in defaults.
  - `fitz config get <key>` — print the value of a config key for the current repo.
    - Example: `fitz config get model`
    - Example: `fitz config --global get model`
  - `fitz config set <key> <value>` — set a config key.
    - Example: `fitz config set model claude-opus-4-5`
    - Example: `fitz config --global set model gpt-5.3-codex`
  - `fitz config unset <key>` — remove a config key (falls back to global/default).
    - Example: `fitz config unset model`
    - Example: `fitz config --global unset agent`
  - `fitz config list` — list all keys and their values for the current repo.
    - Example: `fitz config list`
    - Example: `fitz config --global list`
  - `fitz config help` — show config usage and available subcommands.
    - Example: `fitz config help`
  - Valid keys: `model` (passed as `--model` to Copilot CLI on every invocation), `agent` (agent framework; only `copilot-cli` supported today).
- `fitz br` — interactive worktree list. Navigate with ↑/↓, press enter to switch worktrees, d to delete (with confirmation), n to create a new worktree, p to publish (push + create PR), or q to quit. The root worktree is shown dimmed and non-actionable.
  - Example: `fitz br`
- `fitz br new [--base <branch>] <name> [prompt...]` — create a new worktree. Optionally set a base branch with `--base`. If a prompt is given (one or more words), copilot launches in the background with `--yolo -p "<prompt>"`.
  - Example: `fitz br new feature-login`
  - Example: `fitz br new --base develop feature-login`
  - Example: `fitz br new feature-login implement user authentication`
  - Example: `fitz br new feature-login "implement user authentication"`
  - Example: `fitz br new --base main feature-login implement user authentication`
- `fitz br go <name>` — switch to an existing worktree.
  - Example: `fitz br go feature-login`
- `fitz br rm <name> [--force]` — remove a worktree and its branch (optionally force removal).
  - Example: `fitz br rm feature-login`
  - Example: `fitz br rm feature-login --force`
- `fitz br rm --all [--force]` — remove all worktrees and their branches.
  - Example: `fitz br rm --all`
  - Example: `fitz br rm --all --force`
- `fitz br list` — interactive worktree list (same as `fitz br`). Navigate with ↑/↓/j/k, press enter to switch worktrees (executes `fitz br go`), d to delete a worktree (with confirmation), n to create a new worktree (prompts for name and action), p to publish (push + create PR), or q/esc/ctrl+c to quit.
  - Example: `fitz br list`
- `fitz br cd <name>` — print the path to a worktree (for shell integration).
  - Example: `fitz br cd feature-login`
- `fitz br publish [name]` — push the current branch to origin and open a pull request via Copilot CLI (uses the `create-pr` skill installed at `~/.agents/skills/create-pr/`). Refuses to run from the default branch or detached `HEAD`. Optionally specify a worktree name; defaults to the current worktree.
  - Example: `fitz br publish`
  - Example: `fitz br publish feature-login`
- `fitz br help` — show br usage and available subcommands.
  - Example: `fitz br help`
- `fitz todo <text>` — add a new todo item for the current repo.
  - Example: `fitz todo "fix the login bug"`
  - Example: `fitz todo remember to update docs`
- `fitz todo list` — interactive TUI: navigate with ↑/↓, press enter to create a worktree from a todo, d to mark done, or add a new todo inline.
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
  config        Get and set configuration values
  help          Show this help message
  todo          Quick per-repo todo list
  update        Update fitz to the latest release
  version       Print version information
```
