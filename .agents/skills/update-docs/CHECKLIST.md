# update-docs checklist

- [ ] Command list in README.md matches `internal/cli/execute.go`.
- [ ] `docs/USAGE.md` command sections match actual behavior.
- [ ] Completion usage is exactly `fitz completion <bash|zsh>`.
- [ ] Update/release asset naming notes are still correct.
- [ ] `go test ./...` passes.
- [ ] Consistency rg checks pass.
