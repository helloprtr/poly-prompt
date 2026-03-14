package translate

import (
	"context"
	"strings"
	"testing"
)

type stubTranslator struct {
	got Request
	out string
}

func (s *stubTranslator) Translate(_ context.Context, req Request) (string, error) {
	s.got = req
	return s.out, nil
}

func TestApplyPolicySkipsEnglishTextForEnglishTarget(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{}
	outcome, err := ApplyPolicy(context.Background(), translator, Request{
		Text:       "Explain this Docker command",
		SourceLang: "auto",
		TargetLang: "en",
	}, ModeAuto)
	if err != nil {
		t.Fatalf("ApplyPolicy() error = %v", err)
	}
	if outcome.Decision != DecisionSkipped {
		t.Fatalf("Decision = %q", outcome.Decision)
	}
	if translator.got.Text != "" {
		t.Fatalf("translator should not be called, got %#v", translator.got)
	}
}

func TestApplyPolicyPreservesCodeTokens(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{out: "Explain `go test ./...` and PRTRPRESERVE_0_TOKEN"}
	outcome, err := ApplyPolicy(context.Background(), translator, Request{
		Text:       "이 명령을 설명해줘 `go test ./...`",
		SourceLang: "ko",
		TargetLang: "en",
	}, ModeAuto)
	if err != nil {
		t.Fatalf("ApplyPolicy() error = %v", err)
	}
	if outcome.Decision != DecisionPartialPreserve {
		t.Fatalf("Decision = %q", outcome.Decision)
	}
	if !strings.Contains(outcome.Text, "`go test ./...`") {
		t.Fatalf("Text = %q", outcome.Text)
	}
}

func TestApplyPolicyForceAlwaysTranslates(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{out: "Hello"}
	outcome, err := ApplyPolicy(context.Background(), translator, Request{
		Text:       "안녕하세요",
		SourceLang: "ko",
		TargetLang: "en",
	}, ModeForce)
	if err != nil {
		t.Fatalf("ApplyPolicy() error = %v", err)
	}
	if outcome.Decision != DecisionTranslated {
		t.Fatalf("Decision = %q", outcome.Decision)
	}
	if translator.got.TargetLang != "en" {
		t.Fatalf("TargetLang = %q", translator.got.TargetLang)
	}
}

func TestApplyPolicyProtectsLearnedTerms(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{out: "Keep PRTRPRESERVE_0_TOKEN and PRTRPRESERVE_1_TOKEN intact"}
	outcome, err := ApplyPolicy(context.Background(), translator, Request{
		Text:           "prtr의 BuildPrompt와 PRTR_TARGET를 설명해줘",
		SourceLang:     "ko",
		TargetLang:     "en",
		ProtectedTerms: []string{"BuildPrompt", "PRTR_TARGET"},
	}, ModeAuto)
	if err != nil {
		t.Fatalf("ApplyPolicy() error = %v", err)
	}
	if outcome.Decision != DecisionPartialPreserve {
		t.Fatalf("Decision = %q", outcome.Decision)
	}
	if !strings.Contains(translator.got.Text, "PRTRPRESERVE_0_TOKEN") && !strings.Contains(translator.got.Text, "PRTRPRESERVE_1_TOKEN") {
		t.Fatalf("translator input = %q", translator.got.Text)
	}
	if !strings.Contains(outcome.Text, "BuildPrompt") || !strings.Contains(outcome.Text, "PRTR_TARGET") {
		t.Fatalf("Text = %q", outcome.Text)
	}
}
