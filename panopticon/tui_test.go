package panopticon

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
