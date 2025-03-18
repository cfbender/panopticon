package panopticon

import (
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"gopkg.in/yaml.v3"
)

type Status int

const (
	Pending Status = iota
	Succeeded
	Failed
)

func (s Status) String() string {
	return [...]string{"Pending", "Succeeded", "Failed"}[s]
}

type result struct {
	duration time.Duration
	status   Status
	job      Command
	output   string
}

type model struct {
	spinner  spinner.Model
	results  map[int]result
	quitting bool
	commands []Command
}

type Command struct {
	ID         int
	Cmd        string   `yaml:"cmd"`
	WatchPaths []string `yaml:"watch_paths"`
}

type Config struct {
	Commands []Command `yaml:"commands"`
}

func loadConfig(path string) (Config, error) {
	var conf Config

	data, err := os.ReadFile(path)
	if err != nil {
		return conf, err
	}

	err = yaml.Unmarshal(data, &conf)
	var commands []Command
	for i, cmd := range conf.Commands {
		// Get absolute path for each watch path
		var paths []string
		for _, watchPath := range cmd.WatchPaths {
			absPath, _ := getAbsolutePath(watchPath)
			paths = append(paths, absPath)
		}

		commands = append(commands, Command{
			ID:         i,
			Cmd:        cmd.Cmd,
			WatchPaths: paths,
		})
	}

	return Config{commands}, err
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
