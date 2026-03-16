package deep_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/helloprtr/poly-prompt/internal/deep"
)

// TestDeepShowcase prints the full DeliveryPrompt for the nil_panic scenario.
// Run with: go test ./internal/deep/... -run TestDeepShowcase -v
// to see what prtr produces for real-world inputs.
func TestDeepShowcase(t *testing.T) {
	source, err := os.ReadFile("testdata/scenarios/nil_panic.go")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	opts := deep.Options{
		Action:     "patch",
		Source:     string(source),
		SourceKind: "clipboard",
		RepoRoot:   t.TempDir(),
	}

	result, err := deep.ExecutePatchRun(context.Background(), opts)
	if err != nil {
		t.Fatalf("ExecutePatchRun: %v", err)
	}

	if result.DeliveryPrompt == "" {
		t.Fatal("expected non-empty DeliveryPrompt")
	}

	t.Logf("\n=== SHOWCASE: nil_panic.go ===\n%s\n=== END ===\n", result.DeliveryPrompt)
}

// TestDeepShowcaseAuthRefactor demonstrates the auth refactor scenario.
func TestDeepShowcaseAuthRefactor(t *testing.T) {
	source, err := os.ReadFile("testdata/scenarios/auth_refactor_source.txt")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	opts := deep.Options{
		Action:     "patch",
		Source:     string(source),
		SourceKind: "clipboard",
		RepoRoot:   t.TempDir(),
	}

	result, err := deep.ExecutePatchRun(context.Background(), opts)
	if err != nil {
		t.Fatalf("ExecutePatchRun: %v", err)
	}

	if result.DeliveryPrompt == "" {
		t.Fatal("expected non-empty DeliveryPrompt")
	}

	t.Logf("\n=== SHOWCASE: auth_refactor ===\n%s\n=== END ===\n", result.DeliveryPrompt)
}

// TestDeepShowcaseAPIMigration demonstrates the API migration scenario.
func TestDeepShowcaseAPIMigration(t *testing.T) {
	source, err := os.ReadFile("testdata/scenarios/api_migration_source.txt")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	opts := deep.Options{
		Action:     "patch",
		Source:     string(source),
		SourceKind: "clipboard",
		RepoRoot:   t.TempDir(),
	}

	result, err := deep.ExecutePatchRun(context.Background(), opts)
	if err != nil {
		t.Fatalf("ExecutePatchRun: %v", err)
	}

	if result.DeliveryPrompt == "" {
		t.Fatal("expected non-empty DeliveryPrompt")
	}

	t.Logf("\n=== SHOWCASE: api_migration ===\n%s\n=== END ===\n", result.DeliveryPrompt)
}

// TestShowcaseRuleBasedVsLLM compares rule-based and LLM-enhanced prompts side by side.
// Requires PRTR_LLM_KEY env var to be set; skips otherwise.
func TestShowcaseRuleBasedVsLLM(t *testing.T) {
	apiKey := os.Getenv("PRTR_LLM_KEY")
	if apiKey == "" {
		t.Skip("set PRTR_LLM_KEY to run LLM showcase")
	}

	source, err := os.ReadFile("testdata/scenarios/nil_panic.go")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	// Rule-based
	ruleOpts := deep.Options{
		Action:     "patch",
		Source:     string(source),
		SourceKind: "clipboard",
		RepoRoot:   t.TempDir(),
	}
	ruleResult, err := deep.ExecutePatchRun(context.Background(), ruleOpts)
	if err != nil {
		t.Fatalf("rule-based run: %v", err)
	}

	// LLM-enhanced
	llmOpts := deep.Options{
		Action:      "patch",
		Source:      string(source),
		SourceKind:  "clipboard",
		RepoRoot:    t.TempDir(),
		LLMProvider: "claude",
		LLMAPIKey:   apiKey,
	}
	llmResult, err := deep.ExecutePatchRun(context.Background(), llmOpts)
	if err != nil {
		t.Fatalf("LLM-enhanced run: %v", err)
	}

	t.Logf("\n=== RULE-BASED ===\n%s\n\n=== LLM-ENHANCED ===\n%s\n", ruleResult.DeliveryPrompt, llmResult.DeliveryPrompt)

	// LLM version should differ from rule-based
	if ruleResult.DeliveryPrompt == llmResult.DeliveryPrompt {
		t.Error("expected LLM-enhanced prompt to differ from rule-based")
	}

	// Both should be non-empty
	if strings.TrimSpace(ruleResult.DeliveryPrompt) == "" {
		t.Error("rule-based prompt is empty")
	}
	if strings.TrimSpace(llmResult.DeliveryPrompt) == "" {
		t.Error("LLM-enhanced prompt is empty")
	}
}
