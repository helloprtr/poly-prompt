//go:build !windows

package main

import (
	"strings"
	"testing"
)

func TestDashboardKeyBindings(t *testing.T) {
	m := newDashboardModel("claude", "active", "fix/main")
	choices := m.KeyChoices()
	keys := map[string]bool{}
	for _, c := range choices {
		keys[c.Key] = true
	}
	for _, want := range []string{"g", "t", "s", "h", "q"} {
		if !keys[want] {
			t.Errorf("missing key binding %q", want)
		}
	}
}

func TestDashboardView(t *testing.T) {
	m := newDashboardModel("claude", "active", "fix/main")
	view := m.View()
	if !strings.Contains(view, "prtr") {
		t.Error("View() should contain 'prtr'")
	}
	if !strings.Contains(view, "claude") {
		t.Error("View() should show target")
	}
}
