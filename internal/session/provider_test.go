package session_test

import (
	"sort"
	"testing"

	"github.com/helloprtr/poly-prompt/internal/session"
)

func TestGetProvider_KnownModels(t *testing.T) {
	for _, name := range []string{"claude", "codex", "gemini"} {
		p, ok := session.GetProvider(name)
		if !ok {
			t.Errorf("GetProvider(%q): expected ok=true", name)
		}
		if len(p.Binaries) == 0 {
			t.Errorf("GetProvider(%q): Binaries must not be empty", name)
		}
		if p.ReadResponse == nil {
			t.Errorf("GetProvider(%q): ReadResponse must not be nil", name)
		}
	}
}

func TestGetProvider_Unknown(t *testing.T) {
	_, ok := session.GetProvider("unknown-model")
	if ok {
		t.Error("GetProvider(unknown): expected ok=false")
	}
}

func TestKnownProviders_ContainsAll(t *testing.T) {
	got := session.KnownProviders()
	want := []string{"claude", "codex", "gemini"}
	if len(got) != len(want) {
		t.Fatalf("KnownProviders: got %v, want %v", got, want)
	}
	// must be sorted
	if !sort.StringsAreSorted(got) {
		t.Errorf("KnownProviders: not sorted: %v", got)
	}
	for i, name := range want {
		if got[i] != name {
			t.Errorf("KnownProviders[%d]: got %q, want %q", i, got[i], name)
		}
	}
}

func TestGetProvider_BinaryOrder(t *testing.T) {
	cases := []struct{ model, wantFirst string }{
		{"claude", "claude"},
		{"codex", "codex"},
		{"gemini", "gemini"},
	}
	for _, tc := range cases {
		p, ok := session.GetProvider(tc.model)
		if !ok {
			t.Fatalf("GetProvider(%q): expected ok=true", tc.model)
		}
		if p.Binaries[0] != tc.wantFirst {
			t.Errorf("GetProvider(%q).Binaries[0] = %q, want %q", tc.model, p.Binaries[0], tc.wantFirst)
		}
	}
}
