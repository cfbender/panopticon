package internal

import (
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var (
	mainStyle = lipgloss.
			NewStyle().
			Margin(1, 2).
			PaddingLeft(1).
			PaddingRight(1).
			BorderStyle(lipgloss.NormalBorder())
	viewportMaxHeight = 20
	viewportMaxWidth  = 60
)

const (
	padding = 2
	offset  = 20
)

func (m model) Init() tea.Cmd {
	log.Println("Starting work...")
	setColors(lipgloss.Color(m.theme.Foreground))
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
				height := min(countLines(i.body), viewportMaxHeight)
				vp := viewport.New(viewportMaxWidth, height)
				vp.SetContent(i.body)

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
			} else if msg.String() == "r" {
				i, _ := m.list.SelectedItem().(item)
				m.currentViewport = nil
				command = func() tea.Msg {
					m.list.SetItem(i.id, item{
						title:           i.title,
						body:            i.body,
						emoji:           i.emoji,
						id:              i.id,
						viewport:        nil,
						viewportVisible: false,
						running:         true,
					})
					log.Println("Executing command:", i.title)
					executeCommand(m, i.id)
					return nil
				}
			}
		}
	case tea.WindowSizeMsg:
		return setSizes(m), nil
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

		command = m.progress.SetPercent(percent)
	}

	var listUpdateCmd tea.Cmd
	m.list, listUpdateCmd = m.list.Update(msg)
	m = setSizes(m)
	return m, tea.Batch(listUpdateCmd, command)
}

func (m model) View() string {
	m = setSizes(m)
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

	width, height, _ := term.GetSize(int(os.Stdout.Fd()))
	h, _ := mainStyle.GetFrameSize()
	mainStyle = mainStyle.MaxWidth(width - h)
	mainStyle = mainStyle.MaxHeight(height)

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

func setSizes(m model) model {
	// Get frame dimensions
	h, v := mainStyle.GetFrameSize()

	// Get terminal size
	width, height, _ := term.GetSize(int(os.Stdout.Fd()))
	if width <= 0 || height <= 0 {
		return m // Don't adjust if we can't get dimensions
	}

	// Calculate usable area
	usableWidth := width - h - padding*2
	usableHeight := height - v - padding*2

	// Set main style
	mainStyle = mainStyle.MaxWidth(width)
	mainStyle = mainStyle.MaxHeight(height)

	// Set components sizes
	m.progress.Width = usableWidth - offset

	if m.currentViewport != nil {
		// Split height between list and viewport
		i, _ := m.list.SelectedItem().(item)
		viewportHeight := min(countLines(i.body), usableHeight-15)
		listHeight := usableHeight - viewportHeight - padding*2

		m.list.SetSize(usableWidth-padding, listHeight)
		m.currentViewport.Width = usableWidth - offset - padding*5
		viewportMaxWidth = usableWidth - offset - padding*5
		m.currentViewport.Height = viewportHeight
		viewportMaxHeight = viewportHeight
	} else {
		// List takes full height
		m.list.SetSize(usableWidth-padding, usableHeight-padding)
	}
	return m
}
