package panopticon

import (
	"os"
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
	return conf, err
}
