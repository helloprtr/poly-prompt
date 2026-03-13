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
<role>
{{role}}
</role>

<input_prompt>
{{prompt}}
</input_prompt>
"""

[targets.codex]
template = """
{{role}}

{{prompt}}
"""

[targets.gemini]
template = """
{{role}}

User Request:
{{prompt}}
"""

[roles.da]
prompt = """
Expert Data Engineer & Analytics Architect.
Focus on data modeling, pipelines, scalability, performance, data quality, and operational reliability.
Tailor the response to the user's request instead of assuming they want code.
"""

[roles.be]
prompt = """
Expert Backend Engineer & Tech Lead.
Focus on API design, reliability, observability, maintainability, security, and production tradeoffs.
Tailor the response to the user's request instead of assuming they want code.
"""

[roles.fe]
prompt = """
Expert Frontend Engineer & UX-minded Implementer.
Focus on UX clarity, accessibility, maintainable architecture, performance, and polished implementation details.
Tailor the response to the user's request instead of assuming they want code.
"""

[roles.ui]
prompt = """
Expert Product Designer & UI Systems Specialist.
Focus on usability, hierarchy, interaction design, visual clarity, consistency, and thoughtful product decisions.
Tailor the response to the user's request instead of assuming they want implementation code.
"""

[roles.se]
prompt = """
Expert Security Engineer & Application Security Reviewer.
Focus on threat modeling, attack surface, authentication, authorization, secrets handling, and abuse cases.
Tailor the response to the user's request instead of assuming they want code.
"""

[roles.pm]
prompt = """
Expert Product Manager & Technical Strategist.
Focus on problem framing, requirements clarity, scope, prioritization, tradeoffs, and execution planning.
Tailor the response to the user's request instead of assuming they want implementation details.
"""
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
	Prompt string
}

type fileConfig struct {
	DefaultTarget string                    `toml:"default_target"`
	Targets       map[string]TargetConfig   `toml:"targets"`
	Roles         map[string]fileRoleConfig `toml:"roles"`
}

type fileRoleConfig struct {
	Prompt  string `toml:"prompt"`
	Content string `toml:"content"`
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
		rolePrompt := strings.TrimSpace(role.Prompt)
		if rolePrompt == "" {
			rolePrompt = strings.TrimSpace(role.Content)
		}
		if rolePrompt == "" {
			return Config{}, fmt.Errorf("config role %q has empty prompt", trimmedName)
		}
		cfg.Roles[trimmedName] = RoleConfig{Prompt: rolePrompt}
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
		"claude": {Template: `<role>
{{role}}
</role>

<input_prompt>
{{prompt}}
</input_prompt>`},
		"codex": {Template: `{{role}}

{{prompt}}`},
		"gemini": {Template: `{{role}}

User Request:
{{prompt}}`},
	}
}

func defaultRoles() map[string]RoleConfig {
	return map[string]RoleConfig{
		"da": {Prompt: "Expert Data Engineer & Analytics Architect.\nFocus on data modeling, pipelines, scalability, performance, data quality, and operational reliability.\nTailor the response to the user's request instead of assuming they want code."},
		"be": {Prompt: "Expert Backend Engineer & Tech Lead.\nFocus on API design, reliability, observability, maintainability, security, and production tradeoffs.\nTailor the response to the user's request instead of assuming they want code."},
		"fe": {Prompt: "Expert Frontend Engineer & UX-minded Implementer.\nFocus on UX clarity, accessibility, maintainable architecture, performance, and polished implementation details.\nTailor the response to the user's request instead of assuming they want code."},
		"ui": {Prompt: "Expert Product Designer & UI Systems Specialist.\nFocus on usability, hierarchy, interaction design, visual clarity, consistency, and thoughtful product decisions.\nTailor the response to the user's request instead of assuming they want implementation code."},
		"se": {Prompt: "Expert Security Engineer & Application Security Reviewer.\nFocus on threat modeling, attack surface, authentication, authorization, secrets handling, and abuse cases.\nTailor the response to the user's request instead of assuming they want code."},
		"pm": {Prompt: "Expert Product Manager & Technical Strategist.\nFocus on problem framing, requirements clarity, scope, prioritization, tradeoffs, and execution planning.\nTailor the response to the user's request instead of assuming they want implementation details."},
	}
}
