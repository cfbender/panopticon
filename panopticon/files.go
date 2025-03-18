package panopticon

import (
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

func WatchForChanges(m model, p *tea.Program) {
	for _, cmd := range m.commands {
		watchForChange(cmd, p)
	}
}

func watchForChange(command Command, p *tea.Program) {
	var paths []string
	paths = append(paths, command.WatchPaths...)
	for _, path := range paths {
		subdirs, _ := listSubdirectories(path)
		paths = append(paths, subdirs...)
	}

	log.Println("Watching directories:", paths)

	for _, subdir := range paths {
		go func() {
			log.Println("Adding directory to watch:", subdir)

			watcher, err := fsnotify.NewWatcher()
			if err != nil {
				log.Fatal(err)
			}
			defer watcher.Close()

			err = watcher.Add(subdir)
			if err != nil {
				log.Fatal(err)
			}

			// Wait for a single event then return a message
			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						log.Println("Watcher received bad event")
						os.Exit(1)
					}
					log.Println("event:", event)
					if event.Has(fsnotify.Write) {
						log.Println("modified file:", event.Name)
						runProcess(command, p)
					}
				case watcherErr, ok := <-watcher.Errors:
					if !ok {
						log.Println("Unable to set up file watcher")
						os.Exit(1)
					}
					log.Println("error:", watcherErr)
				}
			}
		}()
	}
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
