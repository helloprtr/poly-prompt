package session

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ClaudeProjectSlug converts a cwd path to the slug Claude Code uses for
// its project directory: replace every "/" with "-".
// Example: "/Users/koo/dev/foo" → "-Users-koo-dev-foo"
func ClaudeProjectSlug(cwd string) string {
	return strings.ReplaceAll(cwd, "/", "-")
}

// FindLatestJSONLInDir returns the path of the most recently modified .jsonl
// file in dir. Returns "" if none found.
func FindLatestJSONLInDir(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var jsonlFiles []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
			jsonlFiles = append(jsonlFiles, e)
		}
	}
	if len(jsonlFiles) == 0 {
		return ""
	}
	sort.Slice(jsonlFiles, func(i, j int) bool {
		ii, _ := jsonlFiles[i].Info()
		ji, _ := jsonlFiles[j].Info()
		if ii == nil || ji == nil {
			return false
		}
		return ii.ModTime().After(ji.ModTime())
	})
	return filepath.Join(dir, jsonlFiles[0].Name())
}

// ReadClaudeResponseFromFile reads a Claude Code JSONL file and returns the
// text content of the last assistant message. Returns "" on any error.
// Note: scanner buffer is capped at 1 MB; lines exceeding this are skipped silently.
func ReadClaudeResponseFromFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	type claudeContent struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type claudeMessage struct {
		Role    string          `json:"role"`
		Content []claudeContent `json:"content"`
	}
	type claudeLine struct {
		Message claudeMessage `json:"message"`
	}

	var lastText string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		var line claudeLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Message.Role != "assistant" {
			continue
		}
		for _, c := range line.Message.Content {
			if c.Type == "text" && strings.TrimSpace(c.Text) != "" {
				lastText = c.Text
				break
			}
		}
	}
	return lastText
}

// ReadCodexResponseFromFile reads a Codex rollout JSONL file and returns the
// last_agent_message from the task_complete event. Returns "" on any error.
// Note: scanner buffer is capped at 1 MB; lines exceeding this are skipped silently.
func ReadCodexResponseFromFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	type codexPayload struct {
		Type             string `json:"type"`
		LastAgentMessage string `json:"last_agent_message"`
	}
	type codexLine struct {
		Type    string       `json:"type"`
		Payload codexPayload `json:"payload"`
	}

	var lastMsg string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		var line codexLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Type == "event_msg" && line.Payload.Type == "task_complete" {
			if strings.TrimSpace(line.Payload.LastAgentMessage) != "" {
				lastMsg = line.Payload.LastAgentMessage
			}
		}
	}
	return lastMsg
}

// ReadClaudeResponse finds the latest Claude Code JSONL for the given cwd and
// returns the last assistant response. claudeProjectsDir defaults to
// ~/.claude/projects when empty. cwd should be os.Getwd(), not the git root.
func ReadClaudeResponse(claudeProjectsDir, cwd string) string {
	if claudeProjectsDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		claudeProjectsDir = filepath.Join(home, ".claude", "projects")
	}
	projectDir := filepath.Join(claudeProjectsDir, ClaudeProjectSlug(cwd))
	jsonlPath := FindLatestJSONLInDir(projectDir)
	if jsonlPath == "" {
		return ""
	}
	return ReadClaudeResponseFromFile(jsonlPath)
}

// ReadCodexResponse finds the latest Codex rollout JSONL and returns the last
// agent message. codexSessionsDir defaults to ~/.codex/sessions when empty.
// Uses filename lexicographic sort (rollout-<ts>-<uuid>.jsonl) — no stat calls needed.
func ReadCodexResponse(codexSessionsDir string) string {
	if codexSessionsDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		codexSessionsDir = filepath.Join(home, ".codex", "sessions")
	}
	latestPath := findLatestCodexRollout(codexSessionsDir)
	if latestPath == "" {
		return ""
	}
	return ReadCodexResponseFromFile(latestPath)
}

// findLatestCodexRollout finds the most recent rollout-*.jsonl by walking
// codexSessionsDir/YYYY/MM/DD/ directories and sorting by filename descending.
// Filename format "rollout-<RFC3339-ts>-<uuid>.jsonl" sorts lexicographically
// by timestamp, so no stat calls are needed.
func findLatestCodexRollout(baseDir string) string {
	var allFiles []string
	_ = filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if strings.HasPrefix(base, "rollout-") && strings.HasSuffix(base, ".jsonl") {
			allFiles = append(allFiles, path)
		}
		return nil
	})
	if len(allFiles) == 0 {
		return ""
	}
	// Sort descending by filename (timestamp prefix guarantees correct order).
	sort.Sort(sort.Reverse(sort.StringSlice(allFiles)))
	return allFiles[0]
}

// GeminiProjectHash computes the project hash Gemini CLI uses for its
// session directory: hex(sha256(cwd)). Mirrors ClaudeProjectSlug.
func GeminiProjectHash(cwd string) string {
	sum := sha256.Sum256([]byte(cwd))
	return hex.EncodeToString(sum[:])
}

// ReadGeminiResponseFromFile reads a Gemini CLI session JSON file and returns
// the text content of the last "gemini"-typed message. Returns "" on any error.
// Only the content field is read; other message fields (e.g. toolCalls) are ignored.
func ReadGeminiResponseFromFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	type geminiMessage struct {
		Type    string `json:"type"`
		Content string `json:"content"`
	}
	type geminiSession struct {
		Messages []geminiMessage `json:"messages"`
	}

	var sess geminiSession
	if err := json.NewDecoder(f).Decode(&sess); err != nil {
		return ""
	}
	var lastText string
	for _, m := range sess.Messages {
		if m.Type == "gemini" && strings.TrimSpace(m.Content) != "" {
			lastText = m.Content
		}
	}
	return lastText
}

// ReadGeminiResponse finds the latest Gemini CLI session for the given cwd and
// returns the last gemini response. geminiDir defaults to ~/.gemini/tmp when empty.
// cwd should be os.Getwd(), not the git root.
func ReadGeminiResponse(geminiDir, cwd string) string {
	if geminiDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		geminiDir = filepath.Join(home, ".gemini", "tmp")
	}
	hash := GeminiProjectHash(cwd)
	chatsDir := filepath.Join(geminiDir, hash, "chats")
	entries, err := os.ReadDir(chatsDir)
	if err != nil {
		return ""
	}
	// Collect session-*.json files and sort descending by name.
	// Filename format: session-<YYYY-MM-DDTHH-MM>-<uuid-prefix>.json
	// Lexicographic descending order = most recent first.
	var sessions []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "session-") && strings.HasSuffix(e.Name(), ".json") {
			sessions = append(sessions, filepath.Join(chatsDir, e.Name()))
		}
	}
	if len(sessions) == 0 {
		return ""
	}
	sort.Sort(sort.Reverse(sort.StringSlice(sessions)))
	return ReadGeminiResponseFromFile(sessions[0])
}
