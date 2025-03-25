package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime/debug"

	panopticon "github.com/cfbender/panopticon/internal"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gobwas/glob"
)

var (
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render
	mainStyle = lipgloss.NewStyle().MarginLeft(1)
)

func main() {
	var (
		runOnStart  bool
		showHelp    bool
		showVersion bool
		verbose     bool
		match       string
		opts        []tea.ProgramOption
	)

	flag.BoolVar(&runOnStart, "run-on-start", false, "whether to run all commands on start")
	flag.BoolVar(&runOnStart, "r", false, "whether to run all commands on start")

	flag.BoolVar(&showHelp, "h", false, "show help")
	flag.BoolVar(&showHelp, "help", false, "show help")

	flag.BoolVar(&verbose, "verbose", false, "log output to pan.log")

	flag.BoolVar(&showVersion, "v", false, "show version")

	flag.StringVar(&match, "match", "*", "glob pattern to match commands")
	flag.StringVar(&match, "m", "*", "glob pattern to match commands")

	flag.Parse()

	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if showVersion {
		info, _ := debug.ReadBuildInfo()
		fmt.Println(info.Main.Version)
		os.Exit(0)
	}

	argsWithoutProg := os.Args[1:]

	if len(argsWithoutProg) > 0 {
		if argsWithoutProg[0] == "init" {
			panopticon.InitConfig()
			os.Exit(0)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := glob.MustCompile(match)
	model := panopticon.NewModel(cancel, g)

	if !verbose {
		log.SetOutput(io.Discard)
	} else {
		f, err := os.OpenFile("pan.log",
			os.O_RDWR|os.O_CREATE|os.O_APPEND,
			0o666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		defer f.Close()

		log.SetOutput(f)
	}

	opts = append(opts, tea.WithAltScreen())
	p := tea.NewProgram(model, opts...)

	if runOnStart {
		go panopticon.RunAll(model, p, ctx)
	}

	go panopticon.WatchForChanges(model, p, ctx)
	go panopticon.WatchForTriggers(model, p, ctx)

	if _, err := p.Run(); err != nil {
		fmt.Println("Error starting Bubble Tea program:", err)
		os.Exit(1)
	}
}
