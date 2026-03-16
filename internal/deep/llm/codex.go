package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/helloprtr/poly-prompt/internal/deep/schema"
)

const codexAPIURL = "https://api.openai.com/v1/chat/completions"
const codexModel = "o4-mini"

var codexHTTPClient = &http.Client{Timeout: 30 * time.Second}

type codexEnhancer struct {
	apiKey  string
	baseURL string
}

func (e *codexEnhancer) Provider() string { return "codex" }

func (e *codexEnhancer) Enhance(ctx context.Context, source string, bundle schema.PatchBundle, ruleBased string) (string, error) {
	url := codexAPIURL
	if e.baseURL != "" {
		url = e.baseURL + "/v1/chat/completions"
	}

	prompt := buildEnhancePrompt(source, bundle, ruleBased)

	reqBody, err := json.Marshal(map[string]any{
		"model": codexModel,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("codex: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("codex: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := codexHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("codex: http: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		if readErr != nil {
			return "", fmt.Errorf("codex: status %d (body unreadable: %w)", resp.StatusCode, readErr)
		}
		return "", fmt.Errorf("codex: status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("codex: parse response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("codex: empty response")
	}
	return result.Choices[0].Message.Content, nil
}
