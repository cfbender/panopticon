package internal

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func runProcess(command Command, p *tea.Program, ctx context.Context) {
	p.Send(result{1, Pending, command, ""})
	var stdout, stderr bytes.Buffer

	cmd := exec.Command("sh", "-c", command.Cmd)
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now() // Start timing here, before command starts
	err := cmd.Start()
	if err != nil {
		p.Send(result{0, Failed, command, err.Error()})
		return
	}

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		killProcess(cmd)
		p.Send(result{time.Since(start), Failed, command, "Command canceled"})
	case err := <-done:
		elapsed := time.Since(start)
		if err != nil {
			p.Send(result{elapsed, Failed, command, stderr.String() + "\n" + stdout.String()})
		} else {
			output := stdout.String()
			if output == "" {
				output = "No output"
			}
			p.Send(result{elapsed, Succeeded, command, output})
		}
	}
}

func RunAll(m model, p *tea.Program, context context.Context) {
	log.Println("Running all commands...")

	for _, cmd := range m.commands {
		go runProcess(cmd, p, context)
	}
}

func killProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	switch runtime.GOOS {
	case "windows":
		killCmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", cmd.Process.Pid))
		return killCmd.Run()

	default: // Linux, macOS, BSD, etc.
		// Set up process group if not already done
		if cmd.SysProcAttr == nil || !cmd.SysProcAttr.Setpgid {
			return fmt.Errorf("process wasn't started with Setpgid=true")
		}

		pgid, err := syscall.Getpgid(cmd.Process.Pid)
		if err != nil {
			return cmd.Process.Kill()
		}

		// Kill the entire process group
		return syscall.Kill(-pgid, syscall.SIGKILL)
	}
}
