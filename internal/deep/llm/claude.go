package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/helloprtr/poly-prompt/internal/deep/schema"
)

const claudeModel = "claude-opus-4-6"
const claudeAPIURL = "https://api.anthropic.com/v1/messages"

type claudeEnhancer struct {
	apiKey  string
	baseURL string // overridable in tests
}

func (e *claudeEnhancer) Provider() string { return "claude" }

func (e *claudeEnhancer) Enhance(ctx context.Context, source string, bundle schema.PatchBundle, ruleBased string) (string, error) {
	url := claudeAPIURL
	if e.baseURL != "" {
		url = e.baseURL + "/v1/messages"
	}

	prompt := buildEnhancePrompt(source, bundle, ruleBased)

	reqBody, err := json.Marshal(map[string]any{
		"model":      claudeModel,
		"max_tokens": 1024,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("claude: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("claude: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", e.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("claude: http: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		if readErr != nil {
			return "", fmt.Errorf("claude: status %d (body unreadable: %w)", resp.StatusCode, readErr)
		}
		return "", fmt.Errorf("claude: status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("claude: parse response: %w", err)
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("claude: empty response")
	}
	return result.Content[0].Text, nil
}
