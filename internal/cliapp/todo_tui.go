package cliapp

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	promptStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("229"))
)

const (
	stateList = iota
	stateBranchInput
	stateActionChoice
)

// TodoAction describes the action the user chose in the TUI.
type TodoAction int

const (
	ActionNone    TodoAction = iota
	ActionGo                 // create worktree + interactive copilot
	ActionKickoff            // create worktree + background copilot with prompt
)

// TodoResult carries the user's selection out of the TUI.
type TodoResult struct {
	Action     TodoAction
	BranchName string
	Prompt     string
}

type todoModel struct {
	items    []TodoItem
	cursor   int
	path     string
	removed  []string
	quitting bool
	state    int

	// branch input state
	selectedTodo TodoItem
	branchInput  textinput.Model

	// action choice state
	actionCursor int

	// inline add todo input (rendered on the last row of the list)
	addInput textinput.Model
	adding   bool

	// dissolve animation state
	dissolving    int // index of item being dissolved, -1 if none
	dissolveFrame int
	dissolveRng   *rand.Rand

	// result
	result TodoResult
}

func newTodoModel(items []TodoItem, path string) todoModel {
	bi := textinput.New()
	bi.Placeholder = "branch-name"
	ai := textinput.New()
	ai.Placeholder = "new todo text"
	return todoModel{items: items, path: path, branchInput: bi, addInput: ai, dissolving: -1}
}

func (m todoModel) Init() tea.Cmd { return nil }

func (m todoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateList:
		return m.updateList(msg)
	case stateBranchInput:
		return m.updateBranchInput(msg)
	case stateActionChoice:
		return m.updateActionChoice(msg)
	}
	return m, nil
}

func (m todoModel) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	// While dissolving, only process tick messages.
	if m.dissolving >= 0 {
		if _, ok := msg.(dissolveTickMsg); ok {
			m.dissolveFrame++
			if m.dissolveFrame > dissolveFrames {
				removed := m.items[m.dissolving]
				m.removed = append(m.removed, removed.Text)
				m.items = append(m.items[:m.dissolving], m.items[m.dissolving+1:]...)
				_ = RemoveTodoItem(m.path, removed.ID)
				if m.cursor >= len(m.items) && m.cursor > 0 {
					m.cursor--
				}
				m.dissolving = -1
				m.dissolveFrame = 0
				m.dissolveRng = nil
				if len(m.items) == 0 {
					m.quitting = true
					return m, tea.Quit
				}
				return m, nil
			}
			return m, dissolveTickCmd()
		}
		return m, nil
	}

	// When adding inline, route input to the text field
	if m.adding {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.adding = false
				m.addInput.Blur()
				return m, nil
			case "enter":
				text := strings.TrimSpace(m.addInput.Value())
				if text == "" {
					m.adding = false
					m.addInput.Blur()
					return m, nil
				}
				_, _ = AddTodoItem(m.path, text)
				items, err := LoadTodos(m.path)
				if err == nil {
					m.items = items
				}
				m.adding = false
				m.addInput.Blur()
				m.addInput.SetValue("")
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.addInput, cmd = m.addInput.Update(msg)
		return m, cmd
	}

	totalRows := len(m.items) + 1 // +1 for "add new" virtual row
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < totalRows-1 {
				m.cursor++
			}
		case "d":
			if m.cursor < len(m.items) && len(m.items) > 0 {
				m.dissolving = m.cursor
				m.dissolveFrame = 1
				m.dissolveRng = rand.New(rand.NewSource(int64(len(m.items[m.cursor].Text))))
				return m, dissolveTickCmd()
			}
		case "enter":
			if m.cursor == len(m.items) {
				// activate inline add input
				m.addInput.SetValue("")
				m.addInput.Focus()
				m.adding = true
				return m, m.addInput.Cursor.BlinkCmd()
			}
			if len(m.items) > 0 {
				m.selectedTodo = m.items[m.cursor]
				m.branchInput = textinput.New()
				m.branchInput.Placeholder = "branch-name"
				m.branchInput.SetValue(slugify(m.selectedTodo.Text))
				m.branchInput.Focus()
				m.state = stateBranchInput
				return m, m.branchInput.Cursor.BlinkCmd()
			}
		}
	}
	return m, nil
}

func (m todoModel) updateBranchInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.state = stateList
			m.branchInput.Blur()
			return m, nil
		case "enter":
			name := strings.TrimSpace(m.branchInput.Value())
			if name == "" {
				return m, nil
			}
			m.result.BranchName = name
			m.actionCursor = 0
			m.state = stateActionChoice
			m.branchInput.Blur()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.branchInput, cmd = m.branchInput.Update(msg)
	return m, cmd
}

func (m todoModel) updateActionChoice(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.branchInput.Focus()
			m.state = stateBranchInput
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
				m.result.Action = ActionGo
			} else {
				m.result.Action = ActionKickoff
				m.result.Prompt = m.selectedTodo.Text
			}
			m.quitting = true
			return m, tea.Quit
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m todoModel) View() string {
	if m.quitting {
		if m.result.Action != ActionNone {
			return ""
		}
		if len(m.removed) > 0 {
			return fmt.Sprintf("Done: %s\n", strings.Join(m.removed, ", "))
		}
		return ""
	}

	switch m.state {
	case stateBranchInput:
		return m.viewBranchInput()
	case stateActionChoice:
		return m.viewActionChoice()
	default:
		return m.viewList()
	}
}

func (m todoModel) viewList() string {
	if len(m.items) == 0 {
		return "No todos.\n"
	}

	var b strings.Builder
	b.WriteString("Todos (↑/↓ navigate, enter create worktree, d done, q quit)\n\n")

	for i, item := range m.items {
		cursor := "  "
		style := dimStyle
		if i == m.cursor {
			cursor = "▸ "
			style = selectedStyle
		}
		displayText := item.Text
		if i == m.dissolving && m.dissolveRng != nil {
			// Create a fresh RNG copy each render so the same frame
			// always produces the same output.
			rng := rand.New(rand.NewSource(m.dissolveRng.Int63()))
			displayText = dissolveText(item.Text, m.dissolveFrame, dissolveFrames, rng)
		}
		b.WriteString(style.Render(fmt.Sprintf("%s%s", cursor, displayText)))
		b.WriteString("\n")
	}

	// virtual "add new" row — inline text input when active
	addCursor := "  "
	addStyle := dimStyle
	if m.cursor == len(m.items) {
		addCursor = "▸ "
		addStyle = selectedStyle
	}
	if m.adding {
		b.WriteString(addStyle.Render(fmt.Sprintf("%s+ ", addCursor)))
		b.WriteString(m.addInput.View())
	} else {
		b.WriteString(addStyle.Render(fmt.Sprintf("%s+ Add new todo...", addCursor)))
	}
	b.WriteString("\n")

	return b.String()
}

func (m todoModel) viewBranchInput() string {
	var b strings.Builder
	b.WriteString(promptStyle.Render(fmt.Sprintf("Create worktree for: %q", m.selectedTodo.Text)))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Branch name: %s", m.branchInput.View()))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("(enter confirm, esc back)"))
	b.WriteString("\n")
	return b.String()
}

func (m todoModel) viewActionChoice() string {
	var b strings.Builder
	b.WriteString(promptStyle.Render(fmt.Sprintf("Create worktree %q for: %q", m.result.BranchName, m.selectedTodo.Text)))
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

func slugify(s string) string {
	s = strings.ToLower(s)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 50 {
		s = s[:50]
		s = strings.TrimRight(s, "-")
	}
	return s
}
