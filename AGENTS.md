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
