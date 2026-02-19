# Fitz: Codebase Analysis & Competitive Gap Analysis

## What Is Fitz?

**Fitz** is a lightweight, terminal-native CLI tool (written in Go) that streamlines agentic coding workflows by wrapping Git worktrees and GitHub Copilot CLI into a single, opinionated developer experience. It turns the manual multi-step process of "create branch → create worktree → launch AI agent → iterate → publish PR" into one-command operations.

### Core Value Proposition

Fitz is a **workflow orchestrator for AI-assisted parallel development**. It sits between the developer and the underlying tools (Git, Copilot CLI, `gh` CLI), abstracting away the complexity of worktree management while adding agentic features like background AI task kickoff and session resumption.

### Design Principles

- **Universal**: Runs in any repo without requiring custom config files or folders in the repo (per `docs/PRINCIPLES.md`).
- **Minimal surface area**: Small, stable v1 command set. No bloat.
- **Terminal-first**: Built for CLI-native developers who prefer shell workflows.
- **Copilot-native**: Deeply integrated with GitHub Copilot CLI as the AI agent backend.

---

## How Fitz Helps With Agentic Coding

### 1. Worktree-Based Parallel Development (`fitz br`)

The `fitz br` command family wraps Git worktrees into a developer-friendly workflow:

| Command | What it does |
|---|---|
| `fitz br new <name> [prompt...]` | Creates a worktree under `~/.fitz/<owner>/<repo>/<name>`, creates a branch, and launches Copilot CLI. If a prompt is given, runs Copilot in `--yolo` mode in the background. |
| `fitz br go <name>` | Switches to a worktree and resumes the most recent Copilot session (by scanning `~/.copilot/session-state/`). |
| `fitz br list` | Interactive TUI (Bubble Tea) showing all worktrees with session status badges (⚡ working, age, summary). |
| `fitz br publish [name]` | Pushes the branch and creates a PR via Copilot CLI using the `create-pr` skill. |
| `fitz br rm` / `fitz br rm --all` | Cleans up worktrees and branches. |

**Key insight**: Fitz stores worktrees in `~/.fitz/<owner>/<repo>/` (not inside the repo), keeping the repo clean and enabling a uniform structure across projects.

### 2. Session Tracking & Resumption

The `internal/session` package scans Copilot's `~/.copilot/session-state/` directory to find the most recent session for each worktree. This enables:

- **Automatic session resume**: `fitz br go` passes `--resume <sessionID>` to Copilot, so the agent picks up where it left off.
- **Status badges in TUI**: The worktree list shows real-time session activity (⚡ working if updated < 2min ago, age otherwise) plus a summary line.

### 3. Background Agent Kickoff

`fitz br new feature-login implement user authentication` creates the worktree and launches `copilot --yolo -p "implement user authentication"` as a detached background process. The developer can continue working elsewhere while the agent works autonomously.

### 4. Todo-to-Worktree Pipeline (`fitz todo`)

The todo system provides a per-repo task list stored at `~/.fitz/<owner>/<repo>/todos.json`. From the interactive TUI, pressing Enter on a todo lets you:

1. Name a branch
2. Choose "Create and go" (interactive Copilot) or "Create and kickoff" (background Copilot with a prompt)

This creates a direct pipeline from **idea → branch → AI agent working on it**.

### 5. PR Publishing (`fitz br publish`)

Pushes the branch and delegates PR creation to Copilot CLI using the `create-pr` skill (installed at `~/.agents/skills/create-pr/`). The skill instructs Copilot to gather context via `git log` and `git diff`, check for PR templates, and create a well-written PR.

---

## Architecture Overview

```
cmd/fitz/          → Entry point (main.go)
internal/
  cli/             → Command routing, arg parsing, Subcommand interface
  cliapp/          → Business logic for all commands
    branch.go      → BrNew, BrGo, BrRemove, BrPublish, BrCd
    br_tui.go      → Interactive worktree list (Bubble Tea)
    commands.go    → Version, Update, Completion
    todo.go        → TodoAdd, TodoList
    todo_tui.go    → Interactive todo list (Bubble Tea)
    todo_store.go  → JSON file-based todo storage
    todo_effect.go → Dissolve animation for TUI deletions
  worktree/        → Git worktree operations (Manager, parsing, validation)
  session/         → Copilot session discovery and metadata parsing
skills/
  create-pr/       → Copilot skill for PR creation
```

**Dependencies**: Go 1.24, Kong (CLI parsing), Bubble Tea + Lip Gloss (TUI), standard library otherwise.

---

## Competitive Landscape

### Direct Competitors (Worktree + AI Agent CLI Tools)

| Tool | Description | Agent Backend | Key Differentiator |
|---|---|---|---|
| **claude-wt** | CLI for managing parallel Claude Code sessions in worktrees | Claude Code | Built specifically for Claude; `uvx claude-wt new` creates worktree + agent session |
| **phantom** | CLI for seamless parallel development with Git worktrees | Agent-agnostic | Focuses on worktree UX; supports fzf, tmux, shell hooks. No AI agent integration. |
| **gtr (Git Worktree Runner)** | CLI that runs commands across multiple worktrees | Agent-agnostic | Emphasizes batch execution across worktrees, tmux integration |
| **Uzi** | Orchestrates multiple AI agents in parallel using worktrees + tmux | Claude Code, Codex, Cursor, Aider | Multi-agent orchestration; supports heterogeneous agent backends |
| **cmux** | Cross-platform desktop app for running and orchestrating multiple AI coding agents in parallel | Claude, OpenAI, Gemini, Codex | Rich Electron UI with diff viewer, cost/token tracking, multi-model A/B testing, crash recovery |
| **Conductor** | Context-driven development tool (Gemini CLI extension + Mac native app) | Gemini, Claude Code, Codex | Persistent project context as Markdown, hierarchical task management (Tracks/Phases/Tasks), automated code reviews |
| **@johnlindquist/worktree** | npm CLI for worktree + PR management | Agent-agnostic | Lightweight, PR integration |

### Broader Agentic Coding Tools

| Tool | Category | Worktree Support | Todo/Task Integration |
|---|---|---|---|
| **Claude Code CLI** | AI coding agent | Manual (user creates worktrees); has `/worktree` custom commands, agent teams | No built-in todo |
| **GitHub Copilot CLI** | AI coding agent | Manual; agent mode supports isolated sessions per worktree | No built-in todo |
| **Aider** | AI pair programming CLI | No native worktree management; works within any directory | No built-in todo |
| **Cursor / Windsurf** | AI-powered IDE | No worktree features; runs in IDE context | IDE-based task tracking |
| **Cline (VS Code)** | VS Code extension for autonomous coding | No worktree features | No built-in todo |
| **Devin AI** | Fully autonomous AI engineer | Cloud-based; own sandbox environments | Cloud task management |
| **todo-cli (JoeyWangTW)** | Terminal task manager with AI agents | No worktree integration | Yes — AI agents complete todos via Gemini/MCP |

### Detailed Comparison: Fitz vs cmux

**cmux** (Coding Agent Multiplexer) is an open-source, cross-platform desktop app (Electron + React) designed for running and orchestrating multiple AI coding agents in parallel.

| Dimension | Fitz | cmux |
|---|---|---|
| **Platform** | CLI-only (Go binary) | Desktop app (Electron + React); Mac, Linux |
| **Agent support** | Copilot CLI only | Claude, OpenAI, Gemini, Codex — multi-model |
| **Worktree isolation** | Automatic (`~/.fitz/<owner>/<repo>/<name>`) | Automatic (local, containerized, or remote via SSH) |
| **Multi-agent orchestration** | One agent per worktree, fire-and-forget | Run multiple agents side-by-side with live monitoring |
| **A/B testing agents** | Not supported | Run different LLMs on the same problem, keep best result |
| **Process monitoring** | Session badges in TUI (⚡ working, age) | Live agent output, per-agent logs, crash recovery |
| **Cost/token tracking** | None | Detailed per-agent and per-workspace resource tracking |
| **Diff/merge tooling** | None (delegates to git) | Built-in diff viewer and verification workflow |
| **UI** | Bubble Tea TUI (terminal) | Rich Electron UI with markdown, mermaid, LaTeX rendering |
| **Todo/task pipeline** | ✅ Built-in per-repo todo list | None |
| **Session resume** | ✅ Automatic Copilot session resume | Manual (workspace state preserved) |
| **PR publishing** | ✅ One-command push + PR via Copilot skill | PR integration available |
| **Zero-config repo** | ✅ No repo-level files required | Requires workspace setup |
| **Install footprint** | Single Go binary (~10MB) | Electron app (~200MB+) |

**Where cmux excels over fitz**: Multi-model support, rich visual UI, live agent monitoring with crash recovery, cost/token tracking, and built-in diff verification. cmux is designed for developers who want a "mission control" dashboard for multiple agents.

**Where fitz excels over cmux**: Lightweight CLI-only footprint, zero-config repo requirement, built-in todo-to-agent pipeline, automatic Copilot session resume, one-command PR publishing. Fitz is optimized for terminal-native developers who want fast, opinionated workflows without leaving the shell.

### Detailed Comparison: Fitz vs Conductor

**Conductor** exists in two forms: (1) a Gemini CLI extension for context-driven development and (2) a Mac native app for multi-agent orchestration with isolated worktree workspaces.

| Dimension | Fitz | Conductor |
|---|---|---|
| **Platform** | CLI-only (Go binary); cross-platform | Gemini CLI extension (cross-platform) + Mac native app |
| **Agent support** | Copilot CLI only | Gemini CLI (extension), Claude Code + Codex (Mac app) |
| **Worktree isolation** | Automatic (`~/.fitz/<owner>/<repo>/<name>`) | Automatic via Mac app; Gemini CLI extension works in current dir |
| **Context management** | None — relies on agent's built-in context | Persistent project context stored as Markdown files in repo (specs, architecture, guidelines) |
| **Task management** | Per-repo todo list (flat, text items) | Hierarchical: Tracks → Phases → Tasks with structured specs and plans |
| **Development workflow** | Create worktree → launch agent → publish PR | Context → Spec & Plan → Implement (structured 3-stage workflow) |
| **Automated reviews** | None | ✅ Gemini-powered automated code review (static analysis, compliance, security scans) |
| **Smart revert** | None (standard git operations) | Logical-unit revert at track/phase/task level |
| **Planning before coding** | None — agent starts immediately | ✅ Enforced spec & plan stage before implementation |
| **Session resume** | ✅ Automatic Copilot session resume | Workspace state preserved in Mac app |
| **PR publishing** | ✅ One-command push + PR via Copilot skill | Integrated in Mac app workflow |
| **Zero-config repo** | ✅ No repo-level files required | Requires Markdown context files committed to repo |
| **Install footprint** | Single Go binary (~10MB) | npm extension (lightweight) + Mac app |

**Where Conductor excels over fitz**: Structured planning-before-coding workflow, persistent project context that ensures agents follow team conventions, hierarchical task management, automated code reviews, and smart logical-unit reverts. Conductor is ideal for teams that need disciplined, auditable AI-assisted development on complex codebases.

**Where fitz excels over Conductor**: Lightweight and universal (no repo-level config files), immediate agent kickoff without planning overhead, cross-platform CLI (vs Mac-only native app), automatic Copilot session resume, and a simpler mental model (worktree = branch = agent). Fitz is better for individual developers who want fast, friction-free parallel workflows without ceremony.

---

## Gap Analysis: Where Fitz Falls Short

### Gap 1: Single-Agent Lock-in (Copilot Only)

**Current state**: Fitz is tightly coupled to GitHub Copilot CLI. All `BrNew`, `BrGo`, and `BrPublish` commands call `copilot` directly.

**Industry trend**: Tools like Uzi, gtr, and phantom are agent-agnostic. Claude-wt supports Claude Code. cmux supports Claude, OpenAI, Gemini, and Codex simultaneously. Conductor supports Gemini and Claude Code. Developers increasingly use multiple agents (Claude Code for complex reasoning, Copilot for quick edits, Aider for specific refactors).

**Impact**: Developers using Claude Code, Aider, Gemini CLI, or Cursor CLI cannot use fitz's core workflow without switching to Copilot. cmux even enables A/B testing the same task across different models.

**Recommendation**: Abstract the agent backend behind an interface. Allow configuration of which agent to launch (e.g., `fitz config agent claude` or `fitz br new --agent aider`).

### Gap 2: No Multi-Agent Orchestration

**Current state**: Fitz launches one agent per worktree in the background. There's no way to monitor, coordinate, or compare multiple running agents.

**Industry trend**: Tools like Uzi and Claude Code's agent teams enable running multiple agents simultaneously with monitoring dashboards, cross-agent messaging, and result comparison. cmux provides a full mission-control dashboard for orchestrating multiple agents across worktrees. Conductor's Mac app manages teams of agents with structured task delegation.

**Impact**: Power users who want to try 3 different implementations of the same feature (a common "imagination" pattern) can't orchestrate that with fitz.

**Recommendation**: Add a `fitz br status` or dashboard command that shows all running agent processes with live status. Consider supporting multi-agent runs for the same task.

### Gap 3: No Agent Process Monitoring

**Current state**: When `fitz br new` launches Copilot in the background, the process is fire-and-forget (`cmd.Start()` with nil stdout/stderr). There's no way to check if the agent is still running, see its output, or know if it failed.

**Industry trend**: Uzi and tmux-based workflows show live agent output in separate panes. Claude-wt tracks session state. VS Code background agents show progress in the sidebar. cmux provides live agent logs, crash recovery, and per-agent cost/token tracking.

**Impact**: After kickoff, the developer has no visibility into what the background agent is doing until they manually check the worktree.

**Recommendation**: Capture agent stdout/stderr to a log file. Add `fitz br logs <name>` to tail the output. Show process status (running/completed/failed) in `fitz br list`.

### Gap 4: No Environment Isolation Beyond Code

**Current state**: Fitz creates isolated worktrees (separate directories and branches) but doesn't handle environment setup — `.env` files, local databases, dev server ports, or dependency installation.

**Industry trend**: Nathan Onn's worktree workflow includes database cloning per branch. Uzi assigns unique ports per worktree. gtr runs setup hooks after worktree creation.

**Impact**: Developers working on projects with environment-specific config must manually copy `.env` files and configure ports for each worktree.

**Recommendation**: Support post-create hooks (e.g., `~/.fitz/hooks/post-create.sh` or a per-repo `.fitz/hooks/` directory — while respecting the "no repo files" principle, this could be opt-in). Copy specified dotfiles automatically.

### Gap 5: No Cross-Repository Support

**Current state**: Fitz operates within a single repository. Each worktree is tied to one repo's Git history.

**Industry trend**: Qodo and multi-repo agentic tools handle coordinated changes across interconnected repositories (e.g., API + frontend + shared library).

**Impact**: Developers working on microservices architectures can't coordinate multi-repo tasks through fitz.

**Recommendation**: This is a larger architectural shift. Consider a `fitz workspace` concept that groups related repos and allows coordinated branch creation across them.

### Gap 6: Limited Session Intelligence

**Current state**: Fitz reads session metadata (ID, cwd, summary, updated_at) from Copilot's YAML files. It uses this for resume and status badges.

**Industry trend**: Claude Code exposes rich session metadata including token usage, tool calls, and completion status. Advanced tools aggregate this into productivity dashboards.

**Impact**: The session badges ("⚡ working", "5m ago") are basic. There's no insight into what the agent accomplished, how many tokens were used, or whether the task succeeded.

**Recommendation**: Parse richer session state if available. Add a `fitz br summary <name>` command that shows the agent's work log. Track task completion status.

### Gap 7: No tmux / Terminal Multiplexer Integration

**Current state**: Fitz switches between worktrees by exec'ing into Copilot. There's no way to view multiple worktrees simultaneously.

**Industry trend**: Uzi, gtr, and phantom integrate with tmux to open each worktree in its own pane/window for side-by-side monitoring.

**Impact**: Developers lose context when switching between worktrees. They can't monitor multiple background agents simultaneously.

**Recommendation**: Add optional tmux integration — `fitz br new --tmux` opens the worktree in a new tmux pane. `fitz br list` could show a tmux-style overview.

### Gap 8: No Notification System

**Current state**: After background kickoff, the developer must manually check on the agent.

**Industry trend**: Background agents in VS Code send notifications when complete. Some CLI tools use OS-level notifications (e.g., `notify-send` on Linux, `osascript` on macOS).

**Impact**: Developers may forget about background tasks or not notice when they complete/fail.

**Recommendation**: Send a desktop notification when a background agent process exits. Optionally integrate with terminal bells or custom webhooks.

### Gap 9: No Conflict Detection or Merge Assistance

**Current state**: Each worktree branch is independent. There's no awareness of potential merge conflicts between parallel branches.

**Industry trend**: Some advanced workflows include periodic rebase/merge checks or conflict pre-detection before branches diverge too far.

**Impact**: When multiple agents work on overlapping code areas in parallel, merge conflicts at PR time can be painful.

**Recommendation**: Add `fitz br check` that runs `git merge-tree` between active branches and the base branch to warn about potential conflicts early.

### Gap 10: No Model/Agent Configuration Per Worktree

**Current state**: The same Copilot CLI invocation is used for every worktree. There's no way to specify different models, temperature settings, or agent configurations per task.

**Industry trend**: Copilot CLI now supports model selection (GPT-4.1, GPT-5 mini). Claude Code supports model switching (Sonnet, Opus). Different tasks benefit from different models.

**Impact**: A quick documentation fix doesn't need the same model/configuration as a complex refactoring task.

**Recommendation**: Allow per-worktree agent configuration, either through flags (`fitz br new --model gpt-5-mini`) or through the todo item metadata.

---

## Fitz's Unique Strengths (Not Found in Competitors)

Despite the gaps, fitz has several unique advantages:

1. **Todo → Branch → Agent pipeline**: No other tool integrates a per-repo todo list that directly spawns worktrees with AI agents. This is a genuinely novel workflow.

2. **Session resume via workspace scanning**: Automatically finding and resuming the right Copilot session when navigating to a worktree (`fitz br go`) is a convenience no other worktree tool provides.

3. **Zero-config repo requirement**: Unlike tools that require `.agent.md`, `.cursor/`, or other repo-level configuration (Conductor requires Markdown context files in the repo), fitz stores everything in `~/.fitz/` and works universally.

4. **One-command PR publishing**: `fitz br publish` handles push + PR creation with AI-generated descriptions in a single step.

5. **Interactive TUI with session awareness**: The Bubble Tea TUI showing worktrees with live session status (⚡ working, age, summary) is more informative than any competitor's CLI list command — though cmux's Electron UI provides richer visual monitoring for users who prefer a GUI.

6. **Install includes agent skills**: The installer sets up the `create-pr` Copilot skill, creating an integrated experience out of the box.

7. **Minimal footprint**: A single ~10MB Go binary vs cmux's ~200MB+ Electron app or Conductor's multi-component install. Fitz respects developers who value lean tooling.

---

## Summary

Fitz occupies a valuable niche as a **lightweight, terminal-native workflow orchestrator for Copilot-powered parallel development**. Its todo-to-agent pipeline and session-aware worktree management are genuinely novel. The main gaps are in agent flexibility (Copilot-only), process monitoring (fire-and-forget background agents), environment isolation, and multi-agent orchestration — areas where tools like Uzi, claude-wt, phantom, cmux, and Conductor have advanced further. cmux in particular highlights the gap in multi-model support and live agent monitoring, while Conductor exposes the lack of structured planning and persistent project context in fitz's workflow. Addressing even a few of these gaps (especially agent-agnostic support and background process monitoring) would significantly strengthen fitz's competitive position.
