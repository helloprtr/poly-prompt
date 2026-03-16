package llm

import (
	"context"
	"fmt"

	"github.com/helloprtr/poly-prompt/internal/deep/schema"
)

// Enhancer takes the rule-based delivery prompt and returns a semantically enriched version.
type Enhancer interface {
	Enhance(ctx context.Context, source string, bundle schema.PatchBundle, ruleBased string) (string, error)
	Provider() string
}

// New creates an Enhancer for the given provider name.
// Supported providers: "claude", "gemini", "codex".
func New(provider, apiKey string) (Enhancer, error) {
	switch provider {
	case "claude", "":
		return &claudeEnhancer{apiKey: apiKey}, nil
	case "gemini":
		return &geminiEnhancer{apiKey: apiKey}, nil
	case "codex", "openai":
		return &codexEnhancer{apiKey: apiKey}, nil
	default:
		return nil, fmt.Errorf("unknown LLM provider %q; supported: claude, gemini, codex", provider)
	}
}

// NewWithBaseURL creates an Enhancer with a custom base URL, intended for tests
// that want to point the enhancer at a local httptest server instead of the
// real provider API.
func NewWithBaseURL(provider, apiKey, baseURL string) (Enhancer, error) {
	switch provider {
	case "claude", "":
		return &claudeEnhancer{apiKey: apiKey, baseURL: baseURL}, nil
	case "gemini":
		return &geminiEnhancer{apiKey: apiKey, baseURL: baseURL}, nil
	case "codex", "openai":
		return &codexEnhancer{apiKey: apiKey, baseURL: baseURL}, nil
	default:
		return nil, fmt.Errorf("unknown LLM provider %q; supported: claude, gemini, codex", provider)
	}
}
