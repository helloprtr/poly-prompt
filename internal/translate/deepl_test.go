package translate

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type stubHTTPDoer struct {
	do func(*http.Request) (*http.Response, error)
}

func (s stubHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	return s.do(req)
}

func TestDeepLClientTranslateSuccess(t *testing.T) {
	t.Parallel()

	client := NewDeepLClient(ClientOptions{
		APIKey:  "test-key",
		BaseURL: "https://example.invalid",
		HTTPClient: stubHTTPDoer{do: func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/v2/translate" {
				t.Fatalf("request path = %q, want %q", r.URL.Path, "/v2/translate")
			}
			if got := r.Header.Get("Authorization"); got != "DeepL-Auth-Key test-key" {
				t.Fatalf("Authorization header = %q", got)
			}
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type header = %q", got)
			}

			var payload requestBody
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if len(payload.Text) != 1 || payload.Text[0] != "안녕하세요" {
				t.Fatalf("payload.Text = %#v", payload.Text)
			}
			if payload.TargetLang != "EN-US" {
				t.Fatalf("payload.TargetLang = %q", payload.TargetLang)
			}
			if payload.SourceLang != "KO" {
				t.Fatalf("payload.SourceLang = %q", payload.SourceLang)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"translations":[{"text":"Hello"}]}`)),
			}, nil
		}},
	})

	got, err := client.Translate(context.Background(), Request{Text: "안녕하세요", SourceLang: "ko", TargetLang: "en"})
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}
	if got != "Hello" {
		t.Fatalf("Translate() = %q, want %q", got, "Hello")
	}
}

func TestDeepLClientTranslateErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		apiKey    string
		do        func(*http.Request) (*http.Response, error)
		wantError string
	}{
		{
			name:      "missing api key",
			apiKey:    "",
			wantError: ErrMissingAPIKey.Error(),
		},
		{
			name:   "non-200 response",
			apiKey: "test-key",
			do: func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader("bad request")),
				}, nil
			},
			wantError: "status 400",
		},
		{
			name:   "request error",
			apiKey: "test-key",
			do: func(*http.Request) (*http.Response, error) {
				return nil, context.DeadlineExceeded
			},
			wantError: "translate request failed",
		},
		{
			name:   "malformed json",
			apiKey: "test-key",
			do: func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"translations":`)),
				}, nil
			},
			wantError: "decode translation response",
		},
		{
			name:   "empty translations",
			apiKey: "test-key",
			do: func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"translations":[]}`)),
				}, nil
			},
			wantError: "did not include any translations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := NewDeepLClient(ClientOptions{
				APIKey:  tt.apiKey,
				BaseURL: "https://example.invalid",
				HTTPClient: stubHTTPDoer{do: func(r *http.Request) (*http.Response, error) {
					if tt.do == nil {
						return nil, errors.New("unexpected request")
					}
					return tt.do(r)
				}},
			})

			_, err := client.Translate(context.Background(), Request{Text: "안녕하세요", TargetLang: "en"})
			if err == nil {
				t.Fatal("Translate() expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("Translate() error = %v, want substring %q", err, tt.wantError)
			}
		})
	}
}
