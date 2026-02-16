---
name: create-pr
description: >
  Create a GitHub pull request with a well-written title and description.
  Use this skill when asked to create a PR and a ~/.fitz folder exists.
---

# Create a Pull Request

When asked to create a pull request for the current branch, follow these steps:

1. **Gather context** — run `git log --oneline main..HEAD` and `git diff main...HEAD --stat` to understand what changed.
2. **Check for PR templates** — look for pull request templates in `.github/PULL_REQUEST_TEMPLATE.md`, `.github/PULL_REQUEST_TEMPLATE/`, and `docs/pull_request_template.md`. If multiple templates exist, choose the one most relevant to the changes. If a template is found, use it as the structure for the PR body and fill in its sections.
3. **Determine the title** — write a concise, imperative-mood title summarising the change (e.g. "Add retry logic to API client"). Do not use the branch name verbatim.
4. **Write the description** — if using a template, fill in its sections. Otherwise include:
   - A short summary of *what* changed and *why*.
   - A bullet list of notable changes if there are multiple commits.
   - Any testing notes (e.g. "run `make test`").
5. **Create the PR** — use the GitHub MCP server's `create_pull_request` tool (or `gh pr create`) with the title and body you wrote. Target the default branch.
6. **Report the result** — output the PR URL.

Keep the description under 300 words. Do not include the full diff in the PR body.
