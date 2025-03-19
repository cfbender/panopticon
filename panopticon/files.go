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

	paths := getPaths(command)

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

func getPaths(command Command) []string {
	var paths []string
	paths = append(paths, command.WatchPaths...)

	ignored := make(map[string]bool, len(command.IgnorePaths))
	for _, path := range command.IgnorePaths {

		absolutePath, err := getAbsolutePath(path)
		if err != nil {
			ignored[absolutePath] = true
		}
	}

	for _, path := range paths {
		absolutePath, err := getAbsolutePath(path)
		if err != nil && !ignored[absolutePath] {
			subdirs, _ := listSubdirectories(path)
			paths = append(paths, subdirs...)
		}
	}
	return paths
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

func getAbsolutePath(relativePath string) (string, error) {
	// If path is already absolute, return it
	if filepath.IsAbs(relativePath) {
		return relativePath, nil
	}

	// Get current working directory
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Join the pwd with the relative path and convert to absolute
	absPath := filepath.Join(pwd, relativePath)

	// Clean the path to remove any ".." or "." segments
	return filepath.Clean(absPath), nil
}

func (m model) closeWatchers() tea.Msg {
	m.cancelAll()
	// sleep 100 ms to allow interrupts
	time.Sleep(100 * time.Millisecond)

	return nil
}
