package internal

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/gobwas/glob"
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
	triggerChans    []chan bool
	quitting        bool
	commands        []Command
	progress        progress.Model
	list            list.Model
	currentViewport *viewport.Model
	currentSelected int
	cancelAll       context.CancelFunc
	theme           Theme
}

type Command struct {
	ID          int
	Cmd         string   `yaml:"cmd" validate:"required"`
	WatchPaths  []string `yaml:"watch_paths" validate:"required"`
	IgnorePaths []string `yaml:"ignore_paths,omitempty"`
}

type Theme struct {
	Foreground string
	Primary    string
	Secondary  string
	Tertiary   string
	Neutral    string
}

type Config struct {
	Commands    []Command `yaml:"commands"`
	ThemePreset string    `yaml:"theme_preset"`
	ThemeConfig Theme     `yaml:"theme"`
}

func NewModel(cancel context.CancelFunc, g glob.Glob, themeOverride string) model {
	config, err := loadConfig(themeOverride)
	if err != nil {
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	var commands []Command
	var i int
	for _, cmd := range config.Commands {
		if g.Match(cmd.Cmd) {
			commands = append(commands, Command{i, cmd.Cmd, cmd.WatchPaths, cmd.IgnorePaths})
			i++
		}
	}

	sp := spinner.New()
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(config.ThemeConfig.Tertiary))
	// Create a slice with one entry per command
	results := make(map[int]result, len(commands))

	// Initialize each result with its corresponding command
	for _, cmd := range commands {
		results[cmd.ID] = result{
			job: cmd,
		}
	}
	items := getDefaultItems(commands)
	list := list.New(items, itemDelegate{}, 0, 0)
	list.Title = "Commands"
	list.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color(config.ThemeConfig.Neutral)).
		Foreground(lipgloss.Color(config.ThemeConfig.Foreground)).
		Padding(0, 1)
	list.SetShowStatusBar(false)
	list.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("ctrl+j", "ctrl+down"),
				key.WithHelp("ctrl+j/ctrl+↓", "scroll down in viewport"),
			),
			key.NewBinding(
				key.WithKeys("ctrl+k", "ctrl+up"),
				key.WithHelp("ctrl+k/ctrl+↑", "scroll up in viewport"),
			),
			key.NewBinding(
				key.WithKeys("r"),
				key.WithHelp("r", "run command now"),
			),
		}
	}
	list.DisableQuitKeybindings()
	list.SetFilteringEnabled(false)
	list.SetShowFilter(false)

	triggerChans := make([]chan bool, len(commands))
	for i := range commands {
		triggerChans[i] = make(chan bool, 1) // Buffered channel
	}

	newModel := model{
		spinner:         sp,
		results:         results,
		commands:        commands,
		progress:        progress.New(progress.WithGradient(config.ThemeConfig.Primary, config.ThemeConfig.Secondary)),
		list:            list,
		currentViewport: nil,
		cancelAll:       cancel,
		triggerChans:    triggerChans,
		theme:           config.ThemeConfig,
	}

	setSizes(newModel)

	return newModel
}

func loadConfig(themeOverride string) (Config, error) {
	// Check if the config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Println("Config file not found, please run panopticon init or create one.")
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

	if themeOverride != "" {
		log.Println("Using theme override:", themeOverride)
		conf.ThemePreset = themeOverride
	}

	if conf.ThemePreset == "" || conf.ThemePreset == "default" {
		conf.ThemePreset = "catppuccin"
		conf.ThemeConfig = catppuccin
	} else {
		switch conf.ThemePreset {
		case "catppuccin":
			conf.ThemeConfig = catppuccin
		case "gruvbox":
			conf.ThemeConfig = gruvbox
		case "dracula":
			conf.ThemeConfig = dracula
		case "solarized":
			conf.ThemeConfig = solarized
		case "nord":
			conf.ThemeConfig = nord
		case "tokyonight":
			conf.ThemeConfig = tokyonight
		default:
			if conf.ThemeConfig == (Theme{}) {
				log.Println("Invalid theme config, using default theme")
				conf.ThemeConfig = catppuccin
			}
		}
	}

	return Config{commands, conf.ThemePreset, conf.ThemeConfig}, err
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
theme: "default"
`)

	log.Println("Creating sample panopticon.yaml")
	return os.WriteFile("panopticon.yaml", content, 0o644)
}
