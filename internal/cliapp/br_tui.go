package cliapp

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"fitz/internal/session"
	"fitz/internal/status"
	"fitz/internal/worktree"
)

// Reuse styles from todo_tui.go (same package).
// var selectedStyle, dimStyle, promptStyle already defined in todo_tui.go

const (
	brStateList = iota
	brStateConfirmDelete
	brStateNewBranch
	brStateNewAction
	brStateNewPrompt
)

// BrAction describes the action the user chose in the TUI.
type BrAction int

const (
	BrActionNone BrAction = iota
	BrActionGo
	BrActionNew
	BrActionNewKickoff
	BrActionPublish
)

// BrResult carries the user's selection out of the TUI.
type BrResult struct {
	Action     BrAction
	Name       string // worktree name for go/publish
	BranchName string // new branch name for new
	Prompt     string // kickoff prompt
}

type brModel struct {
	worktrees []worktree.WorktreeInfo
	current   string // current worktree name
	cursor    int    // index in worktrees (never 0 - root is not selectable)
	state     int
	quitting  bool

	// session info keyed by worktree path
	sessions map[string]session.SessionInfo
	statuses map[string]status.BranchStatus

	// confirm delete state
	confirmName string

	// new branch input state
	branchInput textinput.Model

	// action choice state (go vs kickoff)
	actionCursor int

	// prompt input state (kickoff mode)
	promptInput textinput.Model

	// dissolve animation state
	dissolving    int // index of item being dissolved, -1 if none
	dissolveFrame int
	dissolveRng   *rand.Rand

	// result
	result BrResult

	// callback for removing worktree (allows testing without actual git operations)
	onRemove func(name string) error
}

func newBrModel(worktrees []worktree.WorktreeInfo, current string, sessions map[string]session.SessionInfo) brModel {
	bi := textinput.New()
	bi.Placeholder = "branch-name"
	pi := textinput.New()
	pi.Placeholder = "prompt for copilot"

	// Start cursor at 1 (first non-root worktree).
	cursor := 1
	if len(worktrees) <= 1 {
		cursor = 0 // no non-root worktrees
	}

	return brModel{
		worktrees:   worktrees,
		current:     current,
		cursor:      cursor,
		sessions:    sessions,
		statuses:    map[string]status.BranchStatus{},
		branchInput: bi,
		promptInput: pi,
		dissolving:  -1,
	}
}

func (m brModel) Init() tea.Cmd { return nil }

func (m brModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case brStateList:
		return m.updateList(msg)
	case brStateConfirmDelete:
		return m.updateConfirmDelete(msg)
	case brStateNewBranch:
		return m.updateNewBranch(msg)
	case brStateNewAction:
		return m.updateNewAction(msg)
	case brStateNewPrompt:
		return m.updateNewPrompt(msg)
	}
	return m, nil
}

func (m brModel) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	// While dissolving, only process tick messages.
	if m.dissolving >= 0 {
		if _, ok := msg.(dissolveTickMsg); ok {
			m.dissolveFrame++
			if m.dissolveFrame > dissolveFrames {
				// Animation complete — remove the worktree.
				name := m.worktrees[m.dissolving].Branch
				if name == "" {
					name = m.worktrees[m.dissolving].Name
				}

				// Call removal callback if provided.
				if m.onRemove != nil {
					_ = m.onRemove(name)
				}

				// Remove from list.
				m.worktrees = append(m.worktrees[:m.dissolving], m.worktrees[m.dissolving+1:]...)

				// Adjust cursor if needed.
				if m.cursor >= len(m.worktrees) && m.cursor > 1 {
					m.cursor--
				}
				if m.cursor < 1 && len(m.worktrees) > 1 {
					m.cursor = 1
				}

				m.dissolving = -1
				m.dissolveFrame = 0
				m.dissolveRng = nil
				m.state = brStateList
				return m, nil
			}
			return m, dissolveTickCmd()
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 1 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.worktrees)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.worktrees) <= 1 {
				return m, nil // no non-root worktrees
			}
			// Switch to selected worktree.
			name := m.worktrees[m.cursor].Branch
			if name == "" {
				name = m.worktrees[m.cursor].Name
			}
			m.result.Action = BrActionGo
			m.result.Name = name
			m.quitting = true
			return m, tea.Quit
		case "d":
			if len(m.worktrees) <= 1 {
				return m, nil // no non-root worktrees
			}
			// Start delete confirmation.
			name := m.worktrees[m.cursor].Branch
			if name == "" {
				name = m.worktrees[m.cursor].Name
			}
			m.confirmName = name
			m.state = brStateConfirmDelete
		case "n":
			// Create new worktree.
			m.branchInput.SetValue("")
			m.branchInput.Focus()
			m.state = brStateNewBranch
			return m, m.branchInput.Cursor.BlinkCmd()
		case "p":
			if len(m.worktrees) <= 1 {
				return m, nil // no non-root worktrees
			}
			// Publish selected worktree.
			name := m.worktrees[m.cursor].Branch
			if name == "" {
				name = m.worktrees[m.cursor].Name
			}
			m.result.Action = BrActionPublish
			m.result.Name = name
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m brModel) updateConfirmDelete(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "n":
			m.state = brStateList
			return m, nil
		case "y":
			// Start dissolve animation.
			m.dissolving = m.cursor
			m.dissolveFrame = 1
			m.dissolveRng = rand.New(rand.NewSource(int64(len(m.confirmName))))
			m.state = brStateList
			return m, dissolveTickCmd()
		}
	}
	return m, nil
}

func (m brModel) updateNewBranch(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = brStateList
			m.branchInput.Blur()
			return m, nil
		case "enter":
			name := strings.TrimSpace(m.branchInput.Value())
			if name == "" {
				return m, nil
			}
			m.result.BranchName = name
			m.actionCursor = 0
			m.state = brStateNewAction
			m.branchInput.Blur()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.branchInput, cmd = m.branchInput.Update(msg)
	return m, cmd
}

func (m brModel) updateNewAction(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.branchInput.Focus()
			m.state = brStateNewBranch
			return m, m.branchInput.Cursor.BlinkCmd()
		case "up", "k":
			if m.actionCursor > 0 {
				m.actionCursor--
			}
		case "down", "j":
			if m.actionCursor < 1 {
				m.actionCursor++
			}
		case "enter":
			if m.actionCursor == 0 {
				// "Create and go"
				m.result.Action = BrActionNew
				m.quitting = true
				return m, tea.Quit
			}
			// "Create and kickoff" — transition to prompt input.
			m.promptInput.SetValue("")
			m.promptInput.Focus()
			m.state = brStateNewPrompt
			return m, m.promptInput.Cursor.BlinkCmd()
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m brModel) updateNewPrompt(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = brStateNewAction
			m.promptInput.Blur()
			return m, nil
		case "enter":
			prompt := strings.TrimSpace(m.promptInput.Value())
			if prompt == "" {
				return m, nil
			}
			m.result.Action = BrActionNewKickoff
			m.result.Prompt = prompt
			m.quitting = true
			m.promptInput.Blur()
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.promptInput, cmd = m.promptInput.Update(msg)
	return m, cmd
}

func (m brModel) View() string {
	if m.quitting {
		return ""
	}

	switch m.state {
	case brStateConfirmDelete:
		return m.viewConfirmDelete()
	case brStateNewBranch:
		return m.viewNewBranch()
	case brStateNewAction:
		return m.viewNewAction()
	case brStateNewPrompt:
		return m.viewNewPrompt()
	default:
		return m.viewList()
	}
}

func (m brModel) viewList() string {
	var b strings.Builder

	if len(m.worktrees) <= 1 {
		b.WriteString("No worktrees.\n\n")
		b.WriteString(dimStyle.Render("(n new worktree, q quit)"))
		b.WriteString("\n")
		return b.String()
	}

	b.WriteString("Worktrees (↑/↓ navigate, enter go, d remove, n new, p publish, q quit)\n\n")

	for i, wt := range m.worktrees {
		name := wt.Branch
		if i == 0 {
			name = "root"
		} else if name == "" {
			name = wt.Name
		}

		isCurrent := (i == 0 && m.current == "root") || (i > 0 && m.current == wt.Name)
		cursor := "  "
		style := dimStyle

		if i == 0 {
			// Root is always dimmed and marked with *.
			if isCurrent {
				cursor = "* "
			} else {
				cursor = "  "
			}
			style = dimStyle
		} else {
			// Non-root worktrees.
			if i == m.cursor {
				cursor = "▸ "
				style = selectedStyle
			} else if isCurrent {
				cursor = "* "
				style = lipgloss.NewStyle().Foreground(lipgloss.Color("33")) // cyan
			}
		}

		displayName := name
		if i == m.dissolving && m.dissolveRng != nil {
			rng := rand.New(rand.NewSource(m.dissolveRng.Int63()))
			displayName = dissolveText(name, m.dissolveFrame, dissolveFrames, rng)
		}

		b.WriteString(style.Render(fmt.Sprintf("%s%s", cursor, displayName)))
		if i > 0 {
			if badge := m.sessionBadge(wt); badge != "" {
				b.WriteString(dimStyle.Render("  " + badge))
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m brModel) viewConfirmDelete() string {
	var b strings.Builder
	b.WriteString(promptStyle.Render(fmt.Sprintf("Remove worktree %q and its branch?", m.confirmName)))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("(y confirm, n/esc cancel)"))
	b.WriteString("\n")
	return b.String()
}

func (m brModel) viewNewBranch() string {
	var b strings.Builder
	b.WriteString(promptStyle.Render("Create new worktree"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Branch name: %s", m.branchInput.View()))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("(enter confirm, esc back)"))
	b.WriteString("\n")
	return b.String()
}

func (m brModel) viewNewAction() string {
	var b strings.Builder
	b.WriteString(promptStyle.Render(fmt.Sprintf("Create worktree %q", m.result.BranchName)))
	b.WriteString("\n\n")

	options := []struct {
		label string
		desc  string
	}{
		{"Create and go", "(open Copilot interactively)"},
		{"Create and kickoff", "(run Copilot in background)"},
	}
	for i, opt := range options {
		cursor := "  "
		style := dimStyle
		if i == m.actionCursor {
			cursor = "▸ "
			style = selectedStyle
		}
		b.WriteString(style.Render(fmt.Sprintf("%s%-20s %s", cursor, opt.label, opt.desc)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("(enter select, esc back)"))
	b.WriteString("\n")
	return b.String()
}

func (m brModel) viewNewPrompt() string {
	var b strings.Builder
	b.WriteString(promptStyle.Render(fmt.Sprintf("Kickoff worktree %q", m.result.BranchName)))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Prompt: %s", m.promptInput.View()))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("(enter confirm, esc back)"))
	b.WriteString("\n")
	return b.String()
}

// sessionBadge returns a short status string for the given worktree path,
// or "" if no session exists for it.
func (m brModel) sessionBadge(wt worktree.WorktreeInfo) string {
	parts := make([]string, 0, 3)
	branch := wt.Branch
	if branch == "" {
		branch = wt.Name
	}

	st := m.statuses[branch]
	if st.PRURL != "" {
		parts = append(parts, formatPRBadge(st.PRURL))
	}

	info, hasSession := m.sessions[wt.Path]
	age := time.Duration(0)
	if hasSession && info.SessionID != "" {
		if info.UpdatedAt.IsZero() {
			parts = append(parts, "· session exists")
		} else {
			age = time.Since(info.UpdatedAt)
			if age < 2*time.Minute {
				parts = append(parts, "⚡ working")
			} else {
				parts = append(parts, "· "+formatAge(age)+" ago")
			}
		}
	}

	if st.Message != "" {
		parts = append(parts, st.Message)
	} else if hasSession && info.SessionID != "" && info.Summary != "" && !info.UpdatedAt.IsZero() && age >= 2*time.Minute {
		parts = append(parts, info.Summary)
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "  ")
}

func formatAge(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}

var pullURLPattern = regexp.MustCompile(`/pull/([0-9]+)`)

func formatPRBadge(prURL string) string {
	label := "🔗 PR"
	if m := pullURLPattern.FindStringSubmatch(prURL); len(m) == 2 {
		label = "🔗 PR #" + m[1]
	}
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", prURL, label)
}
