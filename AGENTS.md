# AGENTS

Contributor and agent workflow for this repo:

1. Use TDD: write/adjust a failing test first, then implement the feature/fix, then make tests pass.
2. If your change affects commands, CLI behavior, or docs, run the `.agents/skills/update-docs` skill before committing.
3. Lint before commit:
   ```sh
   make lint
   ```
   This runs formatting checks and `go vet`.
4. Test before commit:
   ```sh
   make test
   ```

Keep changes minimal, and do not commit until lint and tests are both passing.

## Command conventions
Every command that has its own sub-subcommands **must** implement the `Subcommand` interface (defined in `internal/cli/execute.go`) and be registered in the `subcommands` map. The interface requires a `Help(io.Writer)` method, so the compiler enforces that help is provided. The test `TestAllSubcommandsHandleHelp` further verifies that every registered subcommand responds to `help`, `--help`, and `-h` with usage output. See `brCommand` in `internal/cli/execute.go` for the reference pattern.

## Planning
Keep in mind the principles outlined in `docs/PRINCIPLES.md` when making any plans
