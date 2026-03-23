package session

import "sort"

// ProviderDef describes an AI model provider: the binary candidates to launch
// and a function to read the last response from the model's session log.
type ProviderDef struct {
	Binaries     []string
	ReadResponse func(cwd string) string
}

var providers = map[string]ProviderDef{
	"claude": {
		Binaries:     []string{"claude"},
		ReadResponse: func(cwd string) string { return ReadClaudeResponse("", cwd) },
	},
	"codex": {
		Binaries: []string{"codex"},
		// Codex sessions are not scoped by working directory — cwd is ignored.
		ReadResponse: func(cwd string) string { return ReadCodexResponse("") },
	},
	"gemini": {
		Binaries:     []string{"gemini", "gemini-cli"},
		ReadResponse: func(cwd string) string { return ReadGeminiResponse("", cwd) },
	},
}

// GetProvider returns the ProviderDef for the named model.
// Returns (zero, false) for unknown names.
func GetProvider(name string) (ProviderDef, bool) {
	p, ok := providers[name]
	return p, ok
}

// KnownProviders returns a sorted list of registered model names.
// Intended for future use in --to flag validation (currently Out of Scope).
func KnownProviders() []string {
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
