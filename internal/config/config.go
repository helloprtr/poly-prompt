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
template = "{{prompt}}"

[targets.codex]
template = "{{prompt}}"

[targets.gemini]
template = "{{prompt}}"
`

var ErrConfigExists = errors.New("config already exists")

type Config struct {
	DefaultTarget string
	Targets       map[string]TargetConfig
}

type TargetConfig struct {
	Template string `toml:"template"`
}

type fileConfig struct {
	DefaultTarget string                  `toml:"default_target"`
	Targets       map[string]TargetConfig `toml:"targets"`
}

func Load() (Config, error) {
	cfg := Config{
		Targets: defaultTargets(),
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

func defaultTargets() map[string]TargetConfig {
	return map[string]TargetConfig{
		"claude": {Template: "{{prompt}}"},
		"codex":  {Template: "{{prompt}}"},
		"gemini": {Template: "{{prompt}}"},
	}
}
