package cliapp

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type todoModel struct {
	items    []TodoItem
	cursor   int
	path     string
	removed  []string
	quitting bool
}

func newTodoModel(items []TodoItem, path string) todoModel {
	return todoModel{items: items, path: path}
}

func (m todoModel) Init() tea.Cmd { return nil }

func (m todoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter", " ":
			if len(m.items) > 0 {
				removed := m.items[m.cursor]
				m.removed = append(m.removed, removed.Text)
				m.items = append(m.items[:m.cursor], m.items[m.cursor+1:]...)
				_ = RemoveTodoItem(m.path, removed.ID)
				if m.cursor >= len(m.items) && m.cursor > 0 {
					m.cursor--
				}
				if len(m.items) == 0 {
					m.quitting = true
					return m, tea.Quit
				}
			}
		}
	}
	return m, nil
}

func (m todoModel) View() string {
	if m.quitting {
		if len(m.removed) > 0 {
			return fmt.Sprintf("Done: %s\n", strings.Join(m.removed, ", "))
		}
		return ""
	}

	if len(m.items) == 0 {
		return "No todos.\n"
	}

	var b strings.Builder
	b.WriteString("Todos (↑/↓ navigate, enter/space mark done, q quit)\n\n")

	for i, item := range m.items {
		cursor := "  "
		style := dimStyle
		if i == m.cursor {
			cursor = "▸ "
			style = selectedStyle
		}
		b.WriteString(style.Render(fmt.Sprintf("%s%s", cursor, item.Text)))
		b.WriteString("\n")
	}

	return b.String()
}
