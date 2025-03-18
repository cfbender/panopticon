package panopticon

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func runProcess(command Command, p *tea.Program) {
	parts := strings.Fields(command.Cmd)
	if len(parts) == 0 {
		log.Printf("Could not parse command: %s\n", command.Cmd)
		os.Exit(1)
	}
	p.Send(result{1, Pending, command, ""})

	cmd := exec.Command(parts[0], parts[1:]...)

	log.Printf("Running command: %s\n", command.Cmd)
	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("Error running command: %s\n", command.Cmd)
		p.Send(result{elapsed, Failed, command, err.Error()})
	} else {
		p.Send(result{elapsed, Succeeded, command, ""})
	}
}

func RunAll(m model, p *tea.Program) {
	log.Println("Running all commands...")

	for _, cmd := range m.commands {
		go runProcess(cmd, p)
	}
}
