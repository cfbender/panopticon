package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"pan/panopticon"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render
	mainStyle = lipgloss.NewStyle().MarginLeft(1)
)

func main() {
	var (
		runOnStart bool
		showHelp   bool
		verbose    bool
		opts       []tea.ProgramOption
	)

	flag.BoolVar(&runOnStart, "run-on-start", false, "whether to run all commands on start")
	flag.BoolVar(&showHelp, "h", false, "show help")
	flag.BoolVar(&verbose, "verbose", false, "print a bunch of janky logs")
	flag.Parse()

	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	model := panopticon.NewModel()

	if !verbose {
		log.SetOutput(io.Discard)
	}

	p := tea.NewProgram(model, opts...)

	if runOnStart {
		go panopticon.RunAll(model, p)
	}

	go panopticon.WatchForChanges(model, p)

	if _, err := p.Run(); err != nil {
		fmt.Println("Error starting Bubble Tea program:", err)
		os.Exit(1)
	}
}
