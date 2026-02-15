---
name: update-docs
description: Refresh README.md and docs/USAGE.md after CLI command or docs changes.
---

# Update Docs Skill (fitz)

Use this skill whenever command behavior, command names, usage output, completion behavior, install/update flow, or release asset naming changes.

## Triggers
- Any edit under `internal/cli/**`, `internal/cliapp/**`, or `cmd/fitz/**`.
- Any change to install/update behavior (`install.sh`, updater logic).
- Any PR that adds/removes/renames commands or command arguments.
- Any direct edit to `README.md` or `docs/USAGE.md` that should stay aligned.

## Deterministic update steps
1. Inspect current CLI surface from source:
   - `internal/cli/execute.go` (`Usage:` line and command switch)
   - `internal/cliapp/commands.go` (command argument expectations and completion options)
2. Update `README.md`:
   - Keep command bullets in sync with the real command set.
   - Keep install, completion, and update/release naming notes aligned with implementation.
3. Update `docs/USAGE.md`:
   - Create the file if missing.
   - Ensure one section per supported command (`help`, `version`, `update`, `completion`).
   - Keep examples and argument forms identical to code behavior (especially `completion <bash|zsh>`).
4. Keep wording concise and v1-scoped; do not document deferred commands.

## Validation (required)
Run all checks:

```sh
go test ./...
```

```sh
# Validate command list consistency across code and docs
rg -n "Usage: fitz <help\|version\|update\|completion>" internal/cli/execute.go README.md docs/USAGE.md
rg -n "help version update completion|completion <bash\|zsh>" internal/cliapp/commands.go README.md docs/USAGE.md
```

If any check fails, fix docs and rerun before finishing.
