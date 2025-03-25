package panopticon

import (
	"os"
	"testing"

	"github.com/gobwas/glob"
	"github.com/stretchr/testify/require"
)

const sampleConfig = "commands:\n  - cmd: echo 'hello world'\n    watch_paths: ['*']\n  - cmd: echo 'test'\n    watch_paths: ['./panopticon']\n"

func TestGetEmoji(t *testing.T) {
	// map of Status to string
	expectedMap := map[Status]string{
		Pending:   "⏳",
		Succeeded: "✅",
		Failed:    "❌",
	}

	for status, expected := range expectedMap {
		emoji := getEmoji(status)

		require.Equal(t, expected, emoji)
	}
}

func TestGetStatus(t *testing.T) {
	// map of Status to string
	expectedMap := map[Status]string{
		Succeeded: "finished in",
		Failed:    "failed in",
		Pending:   "running...",
	}

	for status, expected := range expectedMap {
		res := result{
			duration: 0,
			status:   status,
		}
		got := getStatus(res)

		require.Contains(t, got, expected)
	}
}

func TestView(t *testing.T) {
	// Test that the view function returns a string
	m := model{}
	view := m.View()

	require.NotEmpty(t, view)
}

func TestNewModel(t *testing.T) {
	err := os.WriteFile(configFile, []byte(sampleConfig), 0o644)
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(configFile)
	}()

	// Test that the NewModel function returns a model
	cancel := func() {}
	m := NewModel(cancel, glob.MustCompile("*"))

	require.NotNil(t, m)

	// Test that the model filters for the pattern
	m = NewModel(cancel, glob.MustCompile("*hello world*"))

	require.Len(t, m.commands, 1)
	require.Equal(t, "echo 'hello world'", m.commands[0].Cmd)
}
