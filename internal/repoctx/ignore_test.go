package repoctx_test

import (
	"strings"
	"testing"

	"github.com/helloprtr/poly-prompt/internal/repoctx"
)

func TestParseIgnore(t *testing.T) {
	patterns := repoctx.ParseIgnorePatterns(".env\n*.key\n*secret*\n# comment\n\n.env.*")
	tests := []struct {
		path string
		want bool
	}{
		{".env", true},
		{"prod.key", true},
		{"mysecretfile", true},
		{".env.local", true},
		{"main.go", false},
		{"config.toml", false},
	}
	for _, tt := range tests {
		got := repoctx.MatchesIgnore(patterns, tt.path)
		if got != tt.want {
			t.Errorf("MatchesIgnore(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestFilterDiffHunks(t *testing.T) {
	diff := `diff --git a/.env b/.env
+++ b/.env
@@ -1,2 +1,2 @@
-SECRET=old
+SECRET=new
diff --git a/main.go b/main.go
+++ b/main.go
@@ -1,1 +1,1 @@
-fmt.Println("hello")
+fmt.Println("world")
`
	patterns := repoctx.ParseIgnorePatterns(".env")
	got := repoctx.FilterDiffHunks(diff, patterns)
	if strings.Contains(got, "SECRET") {
		t.Error("expected .env hunk to be filtered out")
	}
	if !strings.Contains(got, "fmt.Println") {
		t.Error("expected main.go hunk to be kept")
	}
	if !strings.Contains(got, "[excluded by .prtrignore: .env]") {
		t.Error("expected exclusion note for .env")
	}
}
