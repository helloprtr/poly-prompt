package app

import "strings"

// isUIHeavy reports whether text is likely about UI/UX design topics.
// Reserved for future auto-mode detection in runMain.
func isUIHeavy(text string) bool {
	text = strings.ToLower(text)
	if text == "" {
		return false
	}
	if strings.HasPrefix(text, "ui ") || strings.HasPrefix(text, "ux ") {
		return true
	}
	markers := []string{
		" ui ", " ui\n", "\nui ", "\"ui\"", "'ui'", "(ui)",
		" ux ", " ux\n", "\nux ", "\"ux\"", "'ux'", "(ux)",
		"user interface", "user experience",
		"layout", "screen", "wireframe", "onboarding",
		"landing page", "hierarchy", "spacing",
		"interaction design", "visual design", "component library",
		"design system", "dashboard", "hero section",
		"mobile app", "responsive design",
	}
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

// isCodeHeavy reports whether text is likely about code/engineering topics.
// Reserved for future auto-mode detection in runMain.
func isCodeHeavy(text string) bool {
	text = strings.ToLower(text)
	if text == "" {
		return false
	}
	markers := []string{
		"function", "method", "class", "struct", "interface",
		"variable", "constant", "array", "loop", "conditional",
		"algorithm", "data structure", "database", "query",
		"api", "endpoint", "request", "response",
		"error handling", "exception", "debugging",
		"performance", "optimization", "refactoring",
		"unit test", "integration test", "mock",
		"dependency", "import", "module", "package",
	}
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}
