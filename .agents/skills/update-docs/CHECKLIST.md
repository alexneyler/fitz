# update-docs checklist

- [ ] Command list in README.md matches `internal/cli/execute.go`.
- [ ] `docs/USAGE.md` command sections match actual behavior.
- [ ] Docs wording is succinct and user-facing (no unnecessary implementation details).
- [ ] Completion usage is exactly `fitz completion <bash|zsh>`.
- [ ] Update/release asset naming notes are still correct.
- [ ] `go test ./...` passes.
- [ ] Consistency rg checks pass.
