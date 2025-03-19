package panopticon

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#f4b8e4")).Render
	mainStyle = lipgloss.NewStyle().Margin(1, 2).PaddingLeft(1).PaddingRight(1).BorderStyle(lipgloss.NormalBorder())
)

const (
	padding  = 2
	maxWidth = 80
)

func NewModel(cancel context.CancelFunc) model {
	config, err := loadConfig("./panopticon.yaml")
	if err != nil {
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	var commands []Command
	for i, cmd := range config.Commands {
		commands = append(commands, Command{i, cmd.Cmd, cmd.WatchPaths})
	}

	sp := spinner.New()
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#ca9ee6"))
	// Create a slice with one entry per command
	results := make(map[int]result, len(commands))

	// Initialize each result with its corresponding command
	for _, cmd := range commands {
		results[cmd.ID] = result{
			job: cmd,
		}
	}
	items := getDefaultItems(commands)
	list := list.New(items, itemDelegate{}, 0, 0)
	list.Title = "Commands"
	list.SetShowStatusBar(false)
	list.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("ctrl+j", "ctrl+down"),
				key.WithHelp("ctrl+j/ctrl+↓", "scroll down in viewport"),
			),
			key.NewBinding(
				key.WithKeys("ctrl+k", "ctrl+up"),
				key.WithHelp("ctrl+k/ctrl+↑", "scroll up in viewport"),
			),
		}
	}
	list.DisableQuitKeybindings()

	return model{
		spinner:         sp,
		results:         results,
		commands:        commands,
		progress:        progress.New(progress.WithDefaultGradient()),
		list:            list,
		currentViewport: nil,
		cancelAll:       cancel,
	}
}

func (m model) Init() tea.Cmd {
	log.Println("Starting work...")
	return tea.Batch(
		m.spinner.Tick,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var command tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			command = tea.Sequence(m.closeWatchers, tea.Quit)
		case tea.KeyEnter:
			i, _ := m.list.SelectedItem().(item)
			// toggle viewport
			if m.currentSelected == i.id && (m.currentViewport != nil || i.running) {
				m.currentViewport = nil
				command = m.list.SetItem(i.id, item{
					title:           i.title,
					body:            i.body,
					emoji:           i.emoji,
					id:              i.id,
					viewport:        nil,
					viewportVisible: false,
					running:         i.running,
				})
			} else {
				content := m.results[i.id].output
				height := min(countLines(content), 20)
				vp := viewport.New(80, height)
				vp.SetContent(content)

				m.currentViewport = &vp
				m.currentSelected = i.id
				command = m.list.SetItem(i.id, item{
					title:           i.title,
					body:            i.body,
					emoji:           i.emoji,
					id:              i.id,
					viewport:        &vp,
					viewportVisible: true,
					running:         i.running,
				})
			}
		case tea.KeyCtrlJ, tea.KeyCtrlDown:
			m.currentViewport.LineDown(2)
		case tea.KeyCtrlK, tea.KeyCtrlUp:
			m.currentViewport.LineUp(2)
		case tea.KeyRunes:
			if msg.String() == "q" {
				command = tea.Sequence(m.closeWatchers, tea.Quit)
			}
		}
	case tea.WindowSizeMsg:
		h, v := mainStyle.GetFrameSize()
		m.progress.Width = min(msg.Width-padding*2-4, maxWidth)
		m.list.SetSize(msg.Width-h, (msg.Height-v)/2)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		command = cmd
	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		command = cmd
	case result:
		status := getStatus(msg)
		log.Print(status)
		m.results[msg.job.ID] = msg
		m.list.SetItem(msg.job.ID, item{
			id:      msg.job.ID,
			title:   msg.job.Cmd,
			body:    status + "\n" + msg.output,
			emoji:   getEmoji(msg.status),
			running: msg.status == Pending,
		})

		var completed int
		for _, res := range m.results {
			if res.status != Pending {
				completed++
			}
		}
		percent := float64(completed) / float64(len(m.commands))
		log.Printf("Progress: %d/%d (%.2f%%)\n", completed, len(m.commands), percent*100)

		command = m.progress.SetPercent(percent)
	}

	var listUpdateCmd tea.Cmd
	m.list, listUpdateCmd = m.list.Update(msg)
	return m, tea.Batch(listUpdateCmd, command)
}

func (m model) View() string {
	s := "\n" +
		m.spinner.View() + " Watching 👀...\n\n"

	s += m.progress.View() + "\n\n"

	var items []result
	for _, item := range m.results {
		items = append(items, item)
	}

	// Sort by ID
	sort.Slice(items, func(i, j int) bool {
		return items[i].job.ID < items[j].job.ID
	})

	s += m.list.View() + "\n"

	if m.quitting {
		s += "\n"
	}

	return mainStyle.Render(s)
}

func getEmoji(success Status) string {
	switch success {
	case Pending:
		return "⏳"
	case Failed:
		return "❌"
	default:
		return "✅"
	}
}

func getDefaultItems(items []Command) []list.Item {
	var listItems []list.Item
	for _, i := range items {
		listItems = append(listItems, item{title: i.Cmd, body: "Waiting to run", id: i.ID, running: true})
	}
	return listItems
}

func getStatus(res result) string {
	d := time.Duration.Truncate(res.duration, time.Microsecond)
	switch res.status {
	case Succeeded:
		return fmt.Sprintf("%s %s finished in %s\n", getEmoji(res.status), res.job.Cmd, d)
	case Failed:
		return fmt.Sprintf("%s %s failed in %s\n", getEmoji(res.status), res.job.Cmd, d)
	default:
		return fmt.Sprintf("%s %s running...\n", getEmoji(res.status), res.job.Cmd)
	}
}
