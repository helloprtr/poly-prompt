package template

import (
	"errors"
	"regexp"
	"strings"
)

const Placeholder = "{{prompt}}"
const RolePlaceholder = "{{role}}"
const TargetPlaceholder = "{{target}}"
const ContextPlaceholder = "{{context}}"
const OutputFormatPlaceholder = "{{output_format}}"

type Data struct {
	Prompt       string
	Role         string
	Target       string
	Context      string
	OutputFormat string
}

var emptyXMLBlockPattern = regexp.MustCompile(`(?s)<(?:role|context|task|input_prompt|output_format|target)>\s*</(?:role|context|task|input_prompt|output_format|target)>\n*`)
var repeatedNewlinesPattern = regexp.MustCompile(`\n{3,}`)
var emptyLabelLinePattern = regexp.MustCompile(`(?m)^(Role|Target|Context|Output Format):\s*$\n?`)
var emptyCodeLabelLinePattern = regexp.MustCompile(`(?m)^// (Role|Target|Context|Output Format):\s*$\n?`)

func Render(layout, prompt, role string) (string, error) {
	return RenderData(layout, Data{
		Prompt: prompt,
		Role:   role,
	})
}

func RenderData(layout string, data Data) (string, error) {
	if err := Validate(layout); err != nil {
		return "", err
	}

	replacements := map[string]string{
		Placeholder:             strings.TrimSpace(data.Prompt),
		RolePlaceholder:         strings.TrimSpace(data.Role),
		TargetPlaceholder:       strings.TrimSpace(data.Target),
		ContextPlaceholder:      strings.TrimSpace(data.Context),
		OutputFormatPlaceholder: strings.TrimSpace(data.OutputFormat),
	}

	rendered := layout
	for placeholder, value := range replacements {
		rendered = strings.ReplaceAll(rendered, placeholder, value)
	}

	rendered = emptyXMLBlockPattern.ReplaceAllString(rendered, "")
	rendered = emptyLabelLinePattern.ReplaceAllString(rendered, "")
	rendered = emptyCodeLabelLinePattern.ReplaceAllString(rendered, "")
	rendered = repeatedNewlinesPattern.ReplaceAllString(rendered, "\n\n")

	return strings.TrimSpace(rendered), nil
}

func Validate(layout string) error {
	if strings.TrimSpace(layout) == "" {
		return errors.New("template is empty")
	}
	if !strings.Contains(layout, Placeholder) {
		return errors.New("template must contain {{prompt}}")
	}
	return nil
}
