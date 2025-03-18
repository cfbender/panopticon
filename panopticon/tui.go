package panopticon

import (
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render
	mainStyle = lipgloss.NewStyle().MarginLeft(1)
)

func NewModel() model {
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
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("206"))
	// Create a slice with one entry per command
	results := make(map[int]result, len(commands))

	// Initialize each result with its corresponding command
	for _, cmd := range commands {
		results[cmd.ID] = result{
			job: cmd,
		}
	}

	return model{
		spinner:  sp,
		results:  results,
		commands: commands,
	}
}

func (m model) Init() tea.Cmd {
	log.Println("Starting work...")
	return tea.Batch(
		m.spinner.Tick,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyRunes:
			if msg.String() == "q" {
				return m, tea.Quit
			}
			return m, nil
		default:
			return m, nil
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case result:
		d := time.Duration(msg.duration)
		log.Printf("%s %s finished in %s\n", getEmoji(msg.status), msg.job.Cmd, d)
		m.results[msg.job.ID] = msg
		return m, nil
	default:
		return m, nil
	}
}

func (m model) View() string {
	s := "\n" +
		m.spinner.View() + " Watching 👀...\n\n"

	var items []result
	for _, item := range m.results {
		items = append(items, item)
	}

	// Sort by ID
	sort.Slice(items, func(i, j int) bool {
		return items[i].job.ID < items[j].job.ID
	})

	for _, res := range items {
		if res.duration == 0 {
			s += "........................\n"
		} else {
			d := time.Duration.Truncate(res.duration, time.Microsecond)
			switch res.status {
			case Succeeded:
				s += fmt.Sprintf("%s %s finished in %s\n", getEmoji(res.status), res.job.Cmd, d)
			case Failed:
				s += fmt.Sprintf("%s %s failed in %s\n", getEmoji(res.status), res.job.Cmd, d)
				s += fmt.Sprintf("  %s\n", res.output)
			case Pending:
				s += fmt.Sprintf("%s %s running...\n", getEmoji(res.status), res.job.Cmd)
			}
		}
	}

	s += helpStyle("\nPress q, Esc, or Ctrl+c to exit\n")

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
