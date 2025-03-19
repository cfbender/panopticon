package panopticon

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

func WatchForChanges(m model, p *tea.Program, ctx context.Context) []*fsnotify.Watcher {
	var watchers []*fsnotify.Watcher
	for _, cmd := range m.commands {
		watchers = append(watchers, watchForChange(cmd, p, ctx))
	}
	return watchers
}

func watchForChange(command Command, p *tea.Program, ctx context.Context) *fsnotify.Watcher {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	var paths []string
	paths = append(paths, command.WatchPaths...)
	for _, path := range paths {
		subdirs, _ := listSubdirectories(path)
		paths = append(paths, subdirs...)
	}

	// Add all paths to single watcher
	for _, subdir := range paths {
		err = watcher.Add(subdir)
		if err != nil {
			log.Println("Error watching:", subdir, err)
		}
	}

	// Use a cancellable context for command execution
	cmdCtx, cancelCmd := context.WithCancel(ctx)

	// Only one goroutine per watcher
	go func() {
		defer watcher.Close()

		for {
			select {
			case <-ctx.Done():
				cancelCmd() // Cancel any running command
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) {
					// Cancel previous command and start new one
					cancelCmd()
					cmdCtx, cancelCmd = context.WithCancel(ctx)

					go runProcess(command, p, cmdCtx)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("watcher error:", err)
			}
		}
	}()

	return watcher
}

func listSubdirectories(root string) ([]string, error) {
	var dirs []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && path != root {
			dirs = append(dirs, path)
		}
		return nil
	})

	return dirs, err
}

func (m model) closeWatchers() tea.Msg {
	m.cancelAll()
	// sleep 100 ms to allow interrupts
	time.Sleep(100 * time.Millisecond)

	return nil
}
