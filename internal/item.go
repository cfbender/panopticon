package internal

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("#f2d5cf"))
	viewportStyle     = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#f2d5cf")).Padding(0, 2).Margin(0, 2)
)

type item struct {
	title, body     string
	id              int
	viewport        *viewport.Model
	viewportVisible bool
	running         bool
	emoji           string
}

func (i item) FilterValue() string { return i.title + i.body }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%s %s", i.emoji, i.title)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			display := selectedItemStyle.Render("> " + strings.Join(s, " "))
			if i.viewport != nil && i.viewportVisible && !i.running {
				display += "\n" + renderViewport(i)
			}
			return display
		}
	}

	fmt.Fprint(w, fn(str))
}

func renderViewport(i item) string {
	// Render the viewport to a string
	rendered := i.viewport.View()

	// Apply styles to the rendered content
	rendered = viewportStyle.Render(rendered)

	return rendered
}

func countLines(s string) int {
	// Count newlines and add 1 if text doesn't end with newline
	count := strings.Count(s, "\n")
	if len(s) > 0 && !strings.HasSuffix(s, "\n") {
		count++
	}
	return count
}
