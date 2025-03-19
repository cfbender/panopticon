package panopticon

import (
	"strings"
	"testing"
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

		if emoji != expected {
			t.Errorf("Expected %s, got %s", expected, emoji)
		}
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

		if !strings.Contains(got, expected) {
			t.Errorf("Expected %s, got %s", expected, got)
		}
	}
}
