---
name: review
description: >
  Review the current branch codebase with multiple model-diverse sub-agents,
  validate findings for nitpickiness, and return a final actionable list.
---

# Review Codebase

When asked to review code, run this workflow:

1. **Report progress** — at each phase below, run `fitz agent status "Review: <phase>"` so the user sees live updates.

2. **Review the branch diff**
   - The diff is included in the prompt. Focus on the changed code.
   - Respect any focus prompt if provided.

3. **Run model-diverse reviewers** — report: `fitz agent status "Review: running reviewers"`
   - Launch at least 3 `task` sub-agents with different models (for example: `gpt-5.3-codex`, `claude-sonnet-4.6`, `gemini-3-pro-preview`).
   - Ask each reviewer to return only actionable issues with:
     - severity (`critical|high|medium|low`)
     - file path and line(s)
     - why it is a real issue
     - suggested fix

4. **Run adjudication** — report: `fitz agent status "Review: adjudicating findings"`
   - Launch another sub-agent to review all reviewer findings.
   - Filter out nitpicks, style-only comments, duplicates, and weak claims.
   - Keep only high-confidence, valid findings.

5. **Produce final output** — report: `fitz agent status "Review: finalizing"`
   - Return a single consolidated actionable list.
   - Use this format:
     - `- [severity] path:line - issue summary (why it matters)`
   - If nothing actionable remains, output exactly:
     - `No actionable findings.`
