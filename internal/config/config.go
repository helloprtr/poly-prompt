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

const starterConfig = `deepl_api_key = ""
translation_source_lang = "auto"
translation_target_lang = "en"
default_target = "claude"
default_template_preset = "claude-structured"

[targets.claude]
family = "claude"
default_template_preset = "claude-structured"
translation_target_lang = "en"
default_delivery = "open-copy"

[targets.gemini]
family = "gemini"
default_template_preset = "gemini-stepwise"
translation_target_lang = "en"
default_delivery = "open-copy"

[targets.codex]
family = "codex"
default_template_preset = "codex-implement"
translation_target_lang = "en"
default_delivery = "open-copy"

[launchers.claude]
command = "claude"
args = []
paste_delay_ms = 700
submit_mode = "manual"

[launchers.gemini]
command = "gemini"
args = []
paste_delay_ms = 700
submit_mode = "manual"

[launchers.codex]
command = "codex"
args = []
paste_delay_ms = 800
submit_mode = "manual"

[template_presets.claude-structured]
description = "XML-structured default for Claude."
template = """
<role>
{{role}}
</role>

<context>
{{context}}
</context>

<input_prompt>
{{prompt}}
</input_prompt>

<output_format>
{{output_format}}
</output_format>
"""

[template_presets.gemini-stepwise]
description = "Stepwise reasoning scaffold for Gemini."
template = """
{{role}}

Context:
{{context}}

Follow these steps:
1. Briefly summarize the core requirement.
2. Identify edge cases, bottlenecks, or risks.
3. Provide the most useful response for the user's request.

User Request:
{{prompt}}

Output Format:
{{output_format}}
"""

[template_presets.codex-implement]
description = "Implementation-focused prompt for coding models."
template = """
// Target: {{target}}
// Context: {{context}}
// Output Format: {{output_format}}

{{role}}

{{prompt}}
"""

[roles.be]
prompt = """
Expert Backend Engineer & Tech Lead.
Focus on API design, reliability, observability, maintainability, security, and production tradeoffs.
Tailor the response to the user's request instead of assuming they want code.
"""

[roles.be.targets.claude]
prompt = """
Expert Backend Engineer & Tech Lead.
Focus on reliability, architecture, operational tradeoffs, and clear structured reasoning.
Tailor the response to the user's request instead of assuming they want code.
"""
template_preset = "claude-review"

[roles.be.targets.codex]
prompt = """
Expert Backend Engineer & Tech Lead.
Focus on implementation details, concrete file changes, tests, and safe rollout steps.
Tailor the response to the user's request instead of assuming they want code.
"""
template_preset = "codex-implement"

[roles.be.targets.gemini]
prompt = """
Expert Backend Engineer & Tech Lead.
Focus on analysis, comparison, scalability, and stepwise reasoning.
Tailor the response to the user's request instead of assuming they want code.
"""
template_preset = "gemini-stepwise"

[roles.review]
prompt = """
Expert Reviewer.
Focus on bugs, regressions, risk, correctness, missing tests, and tradeoffs.
Tailor the response to the user's request instead of assuming they want code.
"""

[roles.review.targets.claude]
prompt = """
Expert Reviewer.
Focus on risks, regressions, correctness, tradeoffs, and missing considerations.
Lead with findings before suggestions.
"""
template_preset = "claude-review"

[roles.review.targets.codex]
prompt = """
Expert Reviewer.
Focus on concrete bugs, changed-file implications, missing tests, and actionable fixes.
Lead with findings before suggestions.
"""
template_preset = "codex-review"

[roles.review.targets.gemini]
prompt = """
Expert Reviewer.
Focus on structured comparison, risks, and stepwise analysis of regressions and edge cases.
Lead with findings before suggestions.
"""
template_preset = "gemini-stepwise"

[roles.writer]
prompt = """
Expert Technical Writer.
Focus on clarity, brevity, tone, and preserving the user's actual intent.
"""

[profiles.backend_review]
target = "claude"
role = "be"
template_preset = "claude-review"
translation_target_lang = "en"

[shortcuts.review]
target = "claude"
role = "be"
template_preset = "claude-review"
translation_target_lang = "en"
`

var ErrConfigExists = errors.New("config already exists")

type Config struct {
	APIKey                string
	TranslationSourceLang string
	TranslationTargetLang string
	DefaultTarget         string
	DefaultRole           string
	DefaultTemplatePreset string
	Targets               map[string]TargetConfig
	TemplatePresets       map[string]TemplatePresetConfig
	Roles                 map[string]RoleConfig
	Profiles              map[string]ProfileConfig
	Shortcuts             map[string]ShortcutConfig
	Launchers             map[string]LauncherConfig
	UserPath              string
	ProjectPath           string
	HasUserConfig         bool
	HasProjectConfig      bool
	DefaultTargetSource   string
	DefaultRoleSource     string
	DefaultPresetSource   string
	TranslationSource     string
	TranslationTarget     string
	APIKeySource          string
	LLMProvider           string
	LLMAPIKey             string
}

type TargetConfig struct {
	Family                string `toml:"family"`
	DefaultTemplatePreset string `toml:"default_template_preset"`
	TranslationTargetLang string `toml:"translation_target_lang"`
	DefaultDelivery       string `toml:"default_delivery"`
	Template              string `toml:"template"`
	Description           string `toml:"description"`
}

type TemplatePresetConfig struct {
	Template    string `toml:"template"`
	Description string `toml:"description"`
}

type RoleConfig struct {
	Prompt  string
	Targets map[string]RoleTargetConfig `toml:"targets"`
}

type RoleTargetConfig struct {
	Prompt         string `toml:"prompt"`
	TemplatePreset string `toml:"template_preset"`
}

type ProfileConfig struct {
	Target                string `toml:"target"`
	Role                  string `toml:"role"`
	TemplatePreset        string `toml:"template_preset"`
	TranslationTargetLang string `toml:"translation_target_lang"`
	Context               string `toml:"context"`
	OutputFormat          string `toml:"output_format"`
}

type ShortcutConfig struct {
	Target                string `toml:"target"`
	Role                  string `toml:"role"`
	TemplatePreset        string `toml:"template_preset"`
	TranslationTargetLang string `toml:"translation_target_lang"`
	Context               string `toml:"context"`
	OutputFormat          string `toml:"output_format"`
	Description           string `toml:"description"`
}

type LauncherConfig struct {
	Command      string   `toml:"command"`
	Args         []string `toml:"args"`
	PasteDelayMS int      `toml:"paste_delay_ms"`
	SubmitMode   string   `toml:"submit_mode"`
}

type DefaultsUpdate struct {
	APIKey                *string
	TranslationSourceLang *string
	TranslationTargetLang *string
	DefaultTarget         *string
	DefaultRole           *string
	DefaultTemplatePreset *string
}

type fileConfig struct {
	APIKey                string                          `toml:"deepl_api_key"`
	TranslationSourceLang string                          `toml:"translation_source_lang"`
	TranslationTargetLang string                          `toml:"translation_target_lang"`
	DefaultTarget         string                          `toml:"default_target"`
	DefaultRole           string                          `toml:"default_role"`
	DefaultTemplatePreset string                          `toml:"default_template_preset"`
	LLMProvider           string                          `toml:"llm_provider"`
	LLMAPIKey             string                          `toml:"llm_api_key"`
	Targets               map[string]TargetConfig         `toml:"targets"`
	TemplatePresets       map[string]TemplatePresetConfig `toml:"template_presets"`
	Roles                 map[string]fileRoleConfig       `toml:"roles"`
	Profiles              map[string]ProfileConfig        `toml:"profiles"`
	Shortcuts             map[string]ShortcutConfig       `toml:"shortcuts"`
	Launchers             map[string]fileLauncherConfig   `toml:"launchers"`
}

type fileRoleConfig struct {
	Prompt  string                      `toml:"prompt"`
	Content string                      `toml:"content"`
	Targets map[string]RoleTargetConfig `toml:"targets"`
}

type fileLauncherConfig struct {
	Command      string   `toml:"command"`
	Args         []string `toml:"args"`
	PasteDelayMS *int     `toml:"paste_delay_ms"`
	SubmitMode   string   `toml:"submit_mode"`
}

func Load() (Config, error) {
	cfg := Config{
		Targets:               defaultTargets(),
		TemplatePresets:       defaultTemplatePresets(),
		Roles:                 defaultRoles(),
		Profiles:              defaultProfiles(),
		Shortcuts:             defaultShortcuts(),
		Launchers:             defaultLaunchers(),
		TranslationSourceLang: "auto",
		TranslationTargetLang: "en",
	}

	userPath, err := Path()
	if err != nil {
		return Config{}, err
	}
	cfg.UserPath = userPath

	projectPath, err := ProjectPath()
	if err != nil {
		return Config{}, err
	}
	cfg.ProjectPath = projectPath

	if raw, exists, err := loadFile(userPath); err != nil {
		return Config{}, err
	} else if exists {
		cfg.HasUserConfig = true
		if err := applyFileConfig(&cfg, raw, "user config", true); err != nil {
			return Config{}, err
		}
	}

	if raw, exists, err := loadFile(projectPath); err != nil {
		return Config{}, err
	} else if exists {
		cfg.HasProjectConfig = true
		if err := applyFileConfig(&cfg, raw, "project config", false); err != nil {
			return Config{}, err
		}
	}

	if v := os.Getenv("PRTR_LLM_PROVIDER"); v != "" {
		cfg.LLMProvider = v
	}
	if v := os.Getenv("PRTR_LLM_KEY"); v != "" {
		cfg.LLMAPIKey = v
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

	if err := writeFileAtomic(path, []byte(starterConfig)); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}

	return path, nil
}

func Reset() (string, error) {
	path, err := Path()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create config directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(starterConfig), 0o644); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}
	return path, nil
}

func SaveDefaults(update DefaultsUpdate) (string, error) {
	path, err := Path()
	if err != nil {
		return "", err
	}

	raw := fileConfig{}
	if existing, exists, err := loadFile(path); err != nil {
		return "", err
	} else if exists {
		raw = existing
	}

	if update.APIKey != nil {
		raw.APIKey = strings.TrimSpace(*update.APIKey)
	}
	if update.TranslationSourceLang != nil {
		raw.TranslationSourceLang = strings.TrimSpace(*update.TranslationSourceLang)
	}
	if update.TranslationTargetLang != nil {
		raw.TranslationTargetLang = strings.TrimSpace(*update.TranslationTargetLang)
	}
	if update.DefaultTarget != nil {
		raw.DefaultTarget = strings.TrimSpace(*update.DefaultTarget)
	}
	if update.DefaultRole != nil {
		raw.DefaultRole = strings.TrimSpace(*update.DefaultRole)
	}
	if update.DefaultTemplatePreset != nil {
		raw.DefaultTemplatePreset = strings.TrimSpace(*update.DefaultTemplatePreset)
	}

	data, err := toml.Marshal(raw)
	if err != nil {
		return "", fmt.Errorf("encode config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create config directory: %w", err)
	}
	if err := writeFileAtomic(path, data); err != nil {
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

func ProjectPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}

	current := cwd
	for {
		candidate := filepath.Join(current, ".prtr.toml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect project config: %w", err)
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", nil
		}
		current = parent
	}
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

func ResolveRole(cliRole string, cfg Config) string {
	if role := strings.TrimSpace(cliRole); role != "" {
		return role
	}
	return strings.TrimSpace(cfg.DefaultRole)
}

func ResolveTemplatePreset(cliPreset string, cfg Config, target TargetConfig) string {
	if preset := strings.TrimSpace(cliPreset); preset != "" {
		return preset
	}
	if preset := strings.TrimSpace(cfg.DefaultTemplatePreset); preset != "" {
		return preset
	}
	return strings.TrimSpace(target.DefaultTemplatePreset)
}

func ResolveAPIKey(envAPIKey string, cfg Config) (string, string) {
	if key := strings.TrimSpace(envAPIKey); key != "" {
		return key, "environment"
	}
	if key := strings.TrimSpace(cfg.APIKey); key != "" {
		source := cfg.APIKeySource
		if source == "" {
			source = "config"
		}
		return key, source
	}
	return "", ""
}

func AvailableTargets(cfg Config) []string {
	return sortedKeys(cfg.Targets)
}

func AvailableRoles(cfg Config) []string {
	return sortedKeys(cfg.Roles)
}

func AvailableTemplatePresets(cfg Config) []string {
	return sortedKeys(cfg.TemplatePresets)
}

func AvailableProfiles(cfg Config) []string {
	return sortedKeys(cfg.Profiles)
}

func AvailableShortcuts(cfg Config) []string {
	return sortedKeys(cfg.Shortcuts)
}

func loadFile(path string) (fileConfig, bool, error) {
	if strings.TrimSpace(path) == "" {
		return fileConfig{}, false, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fileConfig{}, false, nil
		}
		return fileConfig{}, false, fmt.Errorf("read config: %w", err)
	}

	var raw fileConfig
	if err := toml.Unmarshal(data, &raw); err != nil {
		return fileConfig{}, false, fmt.Errorf("parse config: %w", err)
	}

	return raw, true, nil
}

func applyFileConfig(cfg *Config, raw fileConfig, source string, includeAPIKey bool) error {
	if includeAPIKey {
		if apiKey := strings.TrimSpace(raw.APIKey); apiKey != "" {
			cfg.APIKey = apiKey
			cfg.APIKeySource = source
		}
	}

	if value := normalizeSourceLanguage(raw.TranslationSourceLang); value != "" {
		cfg.TranslationSourceLang = value
		cfg.TranslationSource = source
	}
	if value := normalizeTargetLanguage(raw.TranslationTargetLang); value != "" {
		cfg.TranslationTargetLang = value
		cfg.TranslationTarget = source
	}

	if v := strings.TrimSpace(raw.LLMProvider); v != "" {
		cfg.LLMProvider = v
	}
	if v := strings.TrimSpace(raw.LLMAPIKey); v != "" {
		cfg.LLMAPIKey = v
	}

	if target := strings.TrimSpace(raw.DefaultTarget); target != "" {
		cfg.DefaultTarget = target
		cfg.DefaultTargetSource = source
	}
	if role := strings.TrimSpace(raw.DefaultRole); role != "" {
		cfg.DefaultRole = role
		cfg.DefaultRoleSource = source
	}
	if preset := strings.TrimSpace(raw.DefaultTemplatePreset); preset != "" {
		cfg.DefaultTemplatePreset = preset
		cfg.DefaultPresetSource = source
	}

	for name, target := range raw.Targets {
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" {
			return errors.New("config target names cannot be empty")
		}
		if target.Template != "" {
			if err := validateTemplate(target.Template, fmt.Sprintf("config target %q", trimmedName)); err != nil {
				return err
			}
		}
		cfg.Targets[trimmedName] = TargetConfig{
			Family:                strings.TrimSpace(target.Family),
			DefaultTemplatePreset: strings.TrimSpace(target.DefaultTemplatePreset),
			TranslationTargetLang: normalizeTargetLanguage(target.TranslationTargetLang),
			DefaultDelivery:       normalizeDelivery(target.DefaultDelivery),
			Template:              target.Template,
			Description:           strings.TrimSpace(target.Description),
		}
	}

	for name, preset := range raw.TemplatePresets {
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" {
			return errors.New("config template preset names cannot be empty")
		}
		if err := validateTemplate(preset.Template, fmt.Sprintf("config template preset %q", trimmedName)); err != nil {
			return err
		}
		cfg.TemplatePresets[trimmedName] = TemplatePresetConfig{
			Template:    preset.Template,
			Description: strings.TrimSpace(preset.Description),
		}
	}

	for name, role := range raw.Roles {
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" {
			return errors.New("config role names cannot be empty")
		}
		rolePrompt := strings.TrimSpace(role.Prompt)
		if rolePrompt == "" {
			rolePrompt = strings.TrimSpace(role.Content)
		}
		if rolePrompt == "" {
			return fmt.Errorf("config role %q has empty prompt", trimmedName)
		}
		roleTargets := map[string]RoleTargetConfig{}
		for targetName, targetConfig := range role.Targets {
			normalizedTargetName := strings.TrimSpace(targetName)
			if normalizedTargetName == "" {
				return fmt.Errorf("config role %q has an empty target override name", trimmedName)
			}
			roleTargets[normalizedTargetName] = RoleTargetConfig{
				Prompt:         strings.TrimSpace(targetConfig.Prompt),
				TemplatePreset: strings.TrimSpace(targetConfig.TemplatePreset),
			}
		}
		cfg.Roles[trimmedName] = RoleConfig{Prompt: rolePrompt, Targets: roleTargets}
	}

	for name, profile := range raw.Profiles {
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" {
			return errors.New("config profile names cannot be empty")
		}
		cfg.Profiles[trimmedName] = normalizeProfile(profile)
	}

	for name, shortcut := range raw.Shortcuts {
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" {
			return errors.New("config shortcut names cannot be empty")
		}
		cfg.Shortcuts[trimmedName] = normalizeShortcut(shortcut)
	}

	for name, launcher := range raw.Launchers {
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" {
			return errors.New("config launcher names cannot be empty")
		}
		cfg.Launchers[trimmedName] = normalizeLauncher(cfg.Launchers[trimmedName], launcher)
	}

	return nil
}

func validateTemplate(templateValue, label string) error {
	if strings.TrimSpace(templateValue) == "" {
		return fmt.Errorf("%s has an empty template", label)
	}
	if !strings.Contains(templateValue, "{{prompt}}") {
		return fmt.Errorf("%s template must contain {{prompt}}", label)
	}
	return nil
}

func normalizeProfile(profile ProfileConfig) ProfileConfig {
	return ProfileConfig{
		Target:                strings.TrimSpace(profile.Target),
		Role:                  strings.TrimSpace(profile.Role),
		TemplatePreset:        strings.TrimSpace(profile.TemplatePreset),
		TranslationTargetLang: normalizeTargetLanguage(profile.TranslationTargetLang),
		Context:               strings.TrimSpace(profile.Context),
		OutputFormat:          strings.TrimSpace(profile.OutputFormat),
	}
}

func normalizeShortcut(shortcut ShortcutConfig) ShortcutConfig {
	return ShortcutConfig{
		Target:                strings.TrimSpace(shortcut.Target),
		Role:                  strings.TrimSpace(shortcut.Role),
		TemplatePreset:        strings.TrimSpace(shortcut.TemplatePreset),
		TranslationTargetLang: normalizeTargetLanguage(shortcut.TranslationTargetLang),
		Context:               strings.TrimSpace(shortcut.Context),
		OutputFormat:          strings.TrimSpace(shortcut.OutputFormat),
		Description:           strings.TrimSpace(shortcut.Description),
	}
}

func normalizeLauncher(base LauncherConfig, launcher fileLauncherConfig) LauncherConfig {
	command := strings.TrimSpace(launcher.Command)
	if command == "" {
		command = strings.TrimSpace(base.Command)
	}

	args := launcher.Args
	if args == nil {
		args = base.Args
	}

	pasteDelay := base.PasteDelayMS
	if launcher.PasteDelayMS != nil {
		pasteDelay = *launcher.PasteDelayMS
	}

	submitMode := normalizeSubmitMode(base.SubmitMode)
	if strings.TrimSpace(launcher.SubmitMode) != "" {
		submitMode = normalizeSubmitMode(launcher.SubmitMode)
	}

	return LauncherConfig{
		Command:      command,
		Args:         args,
		PasteDelayMS: pasteDelay,
		SubmitMode:   submitMode,
	}
}

func defaultTargets() map[string]TargetConfig {
	return map[string]TargetConfig{
		"claude": {
			Family:                "claude",
			DefaultTemplatePreset: "claude-structured",
			TranslationTargetLang: "en",
			DefaultDelivery:       "open-copy",
			Description:           "Anthropic Claude family prompts.",
		},
		"codex": {
			Family:                "codex",
			DefaultTemplatePreset: "codex-implement",
			TranslationTargetLang: "en",
			DefaultDelivery:       "open-copy",
			Description:           "Coding-focused prompt shapes.",
		},
		"gemini": {
			Family:                "gemini",
			DefaultTemplatePreset: "gemini-stepwise",
			TranslationTargetLang: "en",
			DefaultDelivery:       "open-copy",
			Description:           "Google Gemini family prompts.",
		},
	}
}

func defaultTemplatePresets() map[string]TemplatePresetConfig {
	return map[string]TemplatePresetConfig{
		"claude-structured": {
			Description: "XML-structured prompt for analysis and design work.",
			Template: `<role>
{{role}}
</role>

<context>
{{context}}
</context>

<input_prompt>
{{prompt}}
</input_prompt>

<output_format>
{{output_format}}
</output_format>`,
		},
		"claude-review": {
			Description: "Claude review preset for careful critiques and risks.",
			Template: `<role>
{{role}}
</role>

<task>
Review the user's request carefully. Call out risks, tradeoffs, bugs, and missing considerations before suggesting changes.
</task>

<context>
{{context}}
</context>

<input_prompt>
{{prompt}}
</input_prompt>

<output_format>
{{output_format}}
</output_format>`,
		},
		"gemini-stepwise": {
			Description: "Stepwise reasoning preset for Gemini.",
			Template: `{{role}}

Context:
{{context}}

Follow these steps:
1. Briefly summarize the core requirement.
2. Identify edge cases, bottlenecks, or risks.
3. Provide the most useful response for the user's request.

User Request:
{{prompt}}

Output Format:
{{output_format}}`,
		},
		"gemini-scalable": {
			Description: "Gemini preset focused on performance and scalability.",
			Template: `{{role}}

Context:
{{context}}

Focus on performance, scalability, and operational tradeoffs.

User Request:
{{prompt}}

Output Format:
{{output_format}}`,
		},
		"codex-implement": {
			Description: "Implementation-oriented coding preset.",
			Template: `// Target: {{target}}
// Context: {{context}}
// Output Format: {{output_format}}

{{role}}

{{prompt}}`,
		},
		"codex-review": {
			Description: "Code review preset for coding models.",
			Template: `// Target: {{target}}
// Context: {{context}}
// Objective: Review for correctness, regressions, security issues, and missing tests.
// Output Format: {{output_format}}

{{role}}

{{prompt}}`,
		},
	}
}

func defaultRoles() map[string]RoleConfig {
	return map[string]RoleConfig{
		"da": {Prompt: "Expert Data Engineer & Analytics Architect.\nFocus on data modeling, pipelines, scalability, performance, data quality, and operational reliability.\nTailor the response to the user's request instead of assuming they want code."},
		"be": {Prompt: "Expert Backend Engineer & Tech Lead.\nFocus on API design, reliability, observability, maintainability, security, and production tradeoffs.\nTailor the response to the user's request instead of assuming they want code.", Targets: map[string]RoleTargetConfig{
			"claude": {Prompt: "Expert Backend Engineer & Tech Lead.\nFocus on reliability, architecture, operational tradeoffs, and clear structured reasoning.\nTailor the response to the user's request instead of assuming they want code.", TemplatePreset: "claude-review"},
			"codex":  {Prompt: "Expert Backend Engineer & Tech Lead.\nFocus on implementation details, concrete file changes, tests, and safe rollout steps.\nTailor the response to the user's request instead of assuming they want code.", TemplatePreset: "codex-implement"},
			"gemini": {Prompt: "Expert Backend Engineer & Tech Lead.\nFocus on analysis, comparison, scalability, and stepwise reasoning.\nTailor the response to the user's request instead of assuming they want code.", TemplatePreset: "gemini-stepwise"},
		}},
		"fe": {Prompt: "Expert Frontend Engineer & UX-minded Implementer.\nFocus on UX clarity, accessibility, maintainable architecture, performance, and polished implementation details.\nTailor the response to the user's request instead of assuming they want code.", Targets: map[string]RoleTargetConfig{
			"claude": {Prompt: "Expert Frontend Engineer & UX-minded Implementer.\nFocus on architecture, UX tradeoffs, accessibility, and clear structured reasoning.\nTailor the response to the user's request instead of assuming they want code.", TemplatePreset: "claude-structured"},
			"codex":  {Prompt: "Expert Frontend Engineer & UX-minded Implementer.\nFocus on implementation details, component changes, testing, and maintainable code.\nTailor the response to the user's request instead of assuming they want code.", TemplatePreset: "codex-implement"},
			"gemini": {Prompt: "Expert Frontend Engineer & UX-minded Implementer.\nFocus on analysis, hierarchy, performance, and stepwise reasoning.\nTailor the response to the user's request instead of assuming they want code.", TemplatePreset: "gemini-stepwise"},
		}},
		"ui": {Prompt: "Expert Product Designer & UI Systems Specialist.\nFocus on usability, hierarchy, interaction design, visual clarity, consistency, and thoughtful product decisions.\nTailor the response to the user's request instead of assuming they want implementation code."},
		"se": {Prompt: "Expert Security Engineer & Application Security Reviewer.\nFocus on threat modeling, attack surface, authentication, authorization, secrets handling, and abuse cases.\nTailor the response to the user's request instead of assuming they want code."},
		"pm": {Prompt: "Expert Product Manager & Technical Strategist.\nFocus on problem framing, requirements clarity, scope, prioritization, tradeoffs, and execution planning.\nTailor the response to the user's request instead of assuming they want implementation details."},
		"review": {Prompt: "Expert Reviewer.\nFocus on bugs, regressions, risk, correctness, missing tests, and tradeoffs.\nTailor the response to the user's request instead of assuming they want code.", Targets: map[string]RoleTargetConfig{
			"claude": {Prompt: "Expert Reviewer.\nFocus on risks, regressions, correctness, tradeoffs, and missing considerations.\nLead with findings before suggestions.", TemplatePreset: "claude-review"},
			"codex":  {Prompt: "Expert Reviewer.\nFocus on concrete bugs, changed-file implications, missing tests, and actionable fixes.\nLead with findings before suggestions.", TemplatePreset: "codex-review"},
			"gemini": {Prompt: "Expert Reviewer.\nFocus on structured comparison, risks, and stepwise analysis of regressions and edge cases.\nLead with findings before suggestions.", TemplatePreset: "gemini-stepwise"},
		}},
		"writer": {Prompt: "Expert Technical Writer.\nFocus on clarity, brevity, tone, and preserving the user's actual intent."},
	}
}

func defaultProfiles() map[string]ProfileConfig {
	return map[string]ProfileConfig{
		"backend_review":  {Target: "claude", Role: "be", TemplatePreset: "claude-review"},
		"backend_fix":     {Target: "codex", Role: "be", TemplatePreset: "codex-implement"},
		"security_review": {Target: "claude", Role: "se", TemplatePreset: "claude-review"},
		"ui_design":       {Target: "gemini", Role: "ui", TemplatePreset: "gemini-stepwise"},
	}
}

func defaultShortcuts() map[string]ShortcutConfig {
	return map[string]ShortcutConfig{
		"ask": {
			Target:         "claude",
			TemplatePreset: "claude-structured",
			Description:    "General-purpose structured prompt.",
		},
		"review": {
			Target:         "claude",
			Role:           "be",
			TemplatePreset: "claude-review",
			Description:    "Review-oriented prompt.",
		},
		"fix": {
			Target:         "codex",
			Role:           "be",
			TemplatePreset: "codex-implement",
			Description:    "Implementation-oriented prompt.",
		},
		"design": {
			Target:         "gemini",
			Role:           "ui",
			TemplatePreset: "gemini-stepwise",
			Description:    "Design and product thinking prompt.",
		},
	}
}

func defaultLaunchers() map[string]LauncherConfig {
	return map[string]LauncherConfig{
		"claude": {Command: "claude", PasteDelayMS: 700, SubmitMode: "manual"},
		"codex":  {Command: "codex", PasteDelayMS: 800, SubmitMode: "manual"},
		"gemini": {Command: "gemini", PasteDelayMS: 700, SubmitMode: "manual"},
	}
}

func normalizeSourceLanguage(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeTargetLanguage(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeDelivery(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	return value
}

func normalizeSubmitMode(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "manual"
	}
	return value
}

func sortedKeys[T any](items map[string]T) []string {
	names := make([]string, 0, len(items))
	for name := range items {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".config-*.toml.tmp")
	if err != nil {
		return fmt.Errorf("create temp config file: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp config: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("commit config: %w", err)
	}
	return nil
}
