package template

import (
	"errors"
	"regexp"
	"strings"
)

const Placeholder = "{{prompt}}"
const RolePlaceholder = "{{role}}"

var repeatedNewlinesPattern = regexp.MustCompile(`\n{3,}`)
var emptyRoleLinePattern = regexp.MustCompile(`(?m)^Role:\s*$\n?`)
var emptyCodeRoleLinePattern = regexp.MustCompile(`(?m)^// Role:\s*$\n?`)

func Render(layout, prompt, role string) (string, error) {
	if err := Validate(layout); err != nil {
		return "", err
	}

	rendered := strings.ReplaceAll(layout, Placeholder, prompt)
	rendered = strings.ReplaceAll(rendered, RolePlaceholder, strings.TrimSpace(role))
	rendered = strings.ReplaceAll(rendered, "<role></role>\n", "")
	rendered = strings.ReplaceAll(rendered, "\n<role></role>", "")
	rendered = emptyRoleLinePattern.ReplaceAllString(rendered, "")
	rendered = emptyCodeRoleLinePattern.ReplaceAllString(rendered, "")
	rendered = repeatedNewlinesPattern.ReplaceAllString(rendered, "\n\n")

	return rendered, nil
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
