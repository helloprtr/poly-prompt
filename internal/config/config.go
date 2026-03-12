package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

const starterConfig = `default_target = "claude"

[targets.claude]
template = """
<role>{{role}}</role>
<task>
Analyze and respond to the following prompt.
If the prompt involves code, provide a production-ready, clean, and optimized solution.
</task>
<constraints>
- Use TypeScript for web tasks and Go/Python for data engineering tasks unless specified otherwise.
- Explain the 'Why' before the 'How'.
- If there are multiple approaches, brief the pros/cons.
</constraints>

<input_prompt>
{{prompt}}
</input_prompt>

Please respond in English.
"""

[targets.codex]
template = """
// Language: Auto-detect
// Role: {{role}}
// Objective: Efficient, secure, and idiomatic code implementation.
// Context: High-performance data processing and modern web architecture.

{{prompt}}

// Instruction: Provide only the code snippet and essential technical notes.
// No conversational filler.
"""

[targets.gemini]
template = """
You are an {{role}}

Follow these steps to answer:
1. Briefly summarize the core requirement.
2. Identify potential edge cases or data pipeline bottlenecks.
3. Provide the most efficient solution with clear comments.

User Request:
{{prompt}}

Focus on performance and scalability. Answer in English.
"""

[roles.da]
content = "Expert Data Engineer & Analytics Architect"

[roles.be]
content = "Expert Backend Engineer & Tech Lead"

[roles.fe]
content = "Expert Frontend Engineer & UX-minded Implementer"

[roles.ui]
content = "Expert Product Designer & UI Systems Specialist"

[roles.se]
content = "Expert Security Engineer & Application Security Reviewer"

[roles.pm]
content = "Expert Product Manager & Technical Strategist"
`

var ErrConfigExists = errors.New("config already exists")

type Config struct {
	DefaultTarget string
	Targets       map[string]TargetConfig
	Roles         map[string]RoleConfig
}

type TargetConfig struct {
	Template string `toml:"template"`
}

type RoleConfig struct {
	Content string `toml:"content"`
}

type fileConfig struct {
	DefaultTarget string                  `toml:"default_target"`
	Targets       map[string]TargetConfig `toml:"targets"`
	Roles         map[string]RoleConfig   `toml:"roles"`
}

func Load() (Config, error) {
	cfg := Config{
		Targets: defaultTargets(),
		Roles:   defaultRoles(),
	}

	path, err := Path()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var raw fileConfig
	if err := toml.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	cfg.DefaultTarget = strings.TrimSpace(raw.DefaultTarget)
	for name, target := range raw.Targets {
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" {
			return Config{}, errors.New("config target names cannot be empty")
		}
		if strings.TrimSpace(target.Template) == "" {
			return Config{}, fmt.Errorf("config target %q has an empty template", trimmedName)
		}
		cfg.Targets[trimmedName] = TargetConfig{Template: target.Template}
	}

	for name, role := range raw.Roles {
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" {
			return Config{}, errors.New("config role names cannot be empty")
		}
		if strings.TrimSpace(role.Content) == "" {
			return Config{}, fmt.Errorf("config role %q has empty content", trimmedName)
		}
		cfg.Roles[trimmedName] = RoleConfig{Content: strings.TrimSpace(role.Content)}
	}

	return cfg, nil
}

func Init() (string, error) {
	path, err := Path()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create config directory: %w", err)
	}

	if _, err := os.Stat(path); err == nil {
		return path, ErrConfigExists
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect config file: %w", err)
	}

	if err := os.WriteFile(path, []byte(starterConfig), 0o644); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}

	return path, nil
}

func Path() (string, error) {
	base := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}

	return filepath.Join(base, "prtr", "config.toml"), nil
}

func ResolveTarget(cliTarget string, cfg Config, envTarget string) string {
	if target := strings.TrimSpace(cliTarget); target != "" {
		return target
	}

	if target := strings.TrimSpace(cfg.DefaultTarget); target != "" {
		return target
	}

	if target := strings.TrimSpace(envTarget); target != "" {
		return target
	}

	return "claude"
}

func AvailableTargets(cfg Config) []string {
	names := make([]string, 0, len(cfg.Targets))
	for name := range cfg.Targets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func AvailableRoles(cfg Config) []string {
	names := make([]string, 0, len(cfg.Roles))
	for name := range cfg.Roles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func defaultTargets() map[string]TargetConfig {
	return map[string]TargetConfig{
		"claude": {Template: `<role>{{role}}</role>
<task>
Analyze and respond to the following prompt.
If the prompt involves code, provide a production-ready, clean, and optimized solution.
</task>
<constraints>
- Use TypeScript for web tasks and Go/Python for data engineering tasks unless specified otherwise.
- Explain the 'Why' before the 'How'.
- If there are multiple approaches, brief the pros/cons.
</constraints>

<input_prompt>
{{prompt}}
</input_prompt>

Please respond in English.`},
		"codex": {Template: `// Language: Auto-detect
// Role: {{role}}
// Objective: Efficient, secure, and idiomatic code implementation.
// Context: High-performance data processing and modern web architecture.

{{prompt}}

// Instruction: Provide only the code snippet and essential technical notes.
// No conversational filler.`},
		"gemini": {Template: `You are an {{role}}

Follow these steps to answer:
1. Briefly summarize the core requirement.
2. Identify potential edge cases or data pipeline bottlenecks.
3. Provide the most efficient solution with clear comments.

User Request:
{{prompt}}

Focus on performance and scalability. Answer in English.`},
	}
}

func defaultRoles() map[string]RoleConfig {
	return map[string]RoleConfig{
		"da": {Content: "Expert Data Engineer & Analytics Architect"},
		"be": {Content: "Expert Backend Engineer & Tech Lead"},
		"fe": {Content: "Expert Frontend Engineer & UX-minded Implementer"},
		"ui": {Content: "Expert Product Designer & UI Systems Specialist"},
		"se": {Content: "Expert Security Engineer & Application Security Reviewer"},
		"pm": {Content: "Expert Product Manager & Technical Strategist"},
	}
}
