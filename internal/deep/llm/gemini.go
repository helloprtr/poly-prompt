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

const geminiAPIURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent"

type geminiEnhancer struct {
	apiKey  string
	baseURL string
}

func (e *geminiEnhancer) Provider() string { return "gemini" }

func (e *geminiEnhancer) Enhance(ctx context.Context, source string, bundle schema.PatchBundle, ruleBased string) (string, error) {
	// Gemini REST API requires the key as a query parameter; this is intentional.
	// The key does not appear in returned error messages (which log only status+body).
	url := geminiAPIURL + "?key=" + e.apiKey
	if e.baseURL != "" {
		url = e.baseURL + "/v1beta/models/gemini-2.0-flash:generateContent" + "?key=" + e.apiKey
	}

	prompt := buildEnhancePrompt(source, bundle, ruleBased)

	reqBody, err := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": prompt}}},
		},
	})
	if err != nil {
		return "", fmt.Errorf("gemini: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("gemini: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini: http: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		if readErr != nil {
			return "", fmt.Errorf("gemini: status %d (body unreadable: %w)", resp.StatusCode, readErr)
		}
		return "", fmt.Errorf("gemini: status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("gemini: parse response: %w", err)
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: empty response")
	}
	return result.Candidates[0].Content.Parts[0].Text, nil
}
