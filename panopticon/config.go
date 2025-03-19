package panopticon

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	"gopkg.in/yaml.v3"
)

type Status int

const (
	Pending Status = iota
	Succeeded
	Failed
	configFile = "./panopticon.yaml"
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
	spinner         spinner.Model
	results         map[int]result
	quitting        bool
	commands        []Command
	progress        progress.Model
	list            list.Model
	currentViewport *viewport.Model
	currentSelected int
	cancelAll       context.CancelFunc
}

type Command struct {
	ID          int
	Cmd         string   `yaml:"cmd" validate:"required"`
	WatchPaths  []string `yaml:"watch_paths" validate:"required"`
	IgnorePaths []string `yaml:"ignore_paths,omitempty"`
}

type Config struct {
	Commands []Command `yaml:"commands"`
}

func loadConfig() (Config, error) {
	// Check if the config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Println("Config file not found, please run pan init or create one.")
		return Config{}, err
	}

	var conf Config

	data, err := os.ReadFile(configFile)
	if err != nil {
		return conf, err
	}

	err = yaml.Unmarshal(data, &conf)
	var commands []Command
	for i, cmd := range conf.Commands {
		// Get absolute path for each watch path
		var watchPaths []string
		for _, watchPath := range cmd.WatchPaths {
			absPath, _ := getAbsolutePath(watchPath)
			watchPaths = append(watchPaths, absPath)
		}

		// Get absolute path for each ignore path
		var ignorePaths []string
		for _, ignorePath := range cmd.IgnorePaths {
			absPath, _ := getAbsolutePath(ignorePath)
			ignorePaths = append(ignorePaths, absPath)
		}

		commands = append(commands, Command{
			ID:          i,
			Cmd:         cmd.Cmd,
			WatchPaths:  watchPaths,
			IgnorePaths: ignorePaths,
		})
	}

	return Config{commands}, err
}

func InitConfig() error {
	// don't if file exists
	if _, err := os.Stat("panopticon.yaml"); err == nil {
		log.Println("panopticon.yaml already exists")
		return nil
	}

	content := []byte(`# yaml-language-server: $schema=panopticon.schema.json
commands:
  - cmd: "echo 'Hello, World!'"
    watch_paths:
      - ./
`)

	log.Println("Creating sample panopticon.yaml")
	return os.WriteFile("panopticon.yaml", content, 0o644)
}
