---
name: update-fitz
description: >
  Report agent progress back to fitz so branch status stays in sync.
---

# Update Fitz Branch Status

When you finish meaningful work in a fitz worktree, update fitz status:

1. Write a short status message (80 chars max):
   - `fitz agent status "Implemented auth middleware"`
2. If you created or found a PR URL, also record it:
   - `fitz agent status --pr https://github.com/owner/repo/pull/42`
3. If both changed, include both in one call:
   - `fitz agent status --pr https://github.com/owner/repo/pull/42 "Ready for review"`

Use imperative, specific status text. Avoid generic updates like "done" or "working".
