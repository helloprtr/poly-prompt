package template

import "testing"

func TestRenderSubstitutesPrompt(t *testing.T) {
	t.Parallel()

	got, err := Render("Please answer in English:\n{{prompt}}", "Hello", "")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	want := "Please answer in English:\nHello"
	if got != want {
		t.Fatalf("Render() = %q, want %q", got, want)
	}
}

func TestRenderRejectsMissingPlaceholder(t *testing.T) {
	t.Parallel()

	if _, err := Render("static text", "Hello", ""); err == nil {
		t.Fatal("Render() expected an error for missing placeholder")
	}
}

func TestRenderSubstitutesRole(t *testing.T) {
	t.Parallel()

	got, err := Render("Role: {{role}}\n\n{{prompt}}", "Hello", "Expert Backend Engineer & Tech Lead")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	want := "Role: Expert Backend Engineer & Tech Lead\n\nHello"
	if got != want {
		t.Fatalf("Render() = %q, want %q", got, want)
	}
}

func TestRenderRemovesEmptyRoleLines(t *testing.T) {
	t.Parallel()

	got, err := Render("// Role: {{role}}\n\n{{prompt}}", "Hello", "")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	want := "Hello"
	if got != want {
		t.Fatalf("Render() = %q, want %q", got, want)
	}
}

func TestRenderRemovesEmptyCodexRoleLine(t *testing.T) {
	t.Parallel()

	got, err := Render("// Role: {{role}}\n// Objective: Test\n\n{{prompt}}", "Hello", "")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	want := "// Objective: Test\n\nHello"
	if got != want {
		t.Fatalf("Render() = %q, want %q", got, want)
	}
}

func TestRenderSubstitutesXMLRole(t *testing.T) {
	t.Parallel()

	got, err := Render("<role>{{role}}</role>\n<task>{{prompt}}</task>", "Hello", "Expert Backend Engineer & Tech Lead")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	want := "<role>Expert Backend Engineer & Tech Lead</role>\n<task>Hello</task>"
	if got != want {
		t.Fatalf("Render() = %q, want %q", got, want)
	}
}

func TestRenderRemovesEmptyXMLRoleLine(t *testing.T) {
	t.Parallel()

	got, err := Render("<role>{{role}}</role>\n<task>{{prompt}}</task>", "Hello", "")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	want := "<task>Hello</task>"
	if got != want {
		t.Fatalf("Render() = %q, want %q", got, want)
	}
}

func TestRenderRemovesEmptyClaudeRoleLine(t *testing.T) {
	t.Parallel()

	got, err := Render("<role>{{role}}</role>\n<task>Analyze</task>\n\n<input_prompt>\n{{prompt}}\n</input_prompt>", "Hello", "")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	want := "<task>Analyze</task>\n\n<input_prompt>\nHello\n</input_prompt>"
	if got != want {
		t.Fatalf("Render() = %q, want %q", got, want)
	}
}

func TestRenderSubstitutesGeminiRoleLine(t *testing.T) {
	t.Parallel()

	got, err := Render("You are an {{role}}\n\nUser Request:\n{{prompt}}", "Hello", "Expert Backend Engineer & Tech Lead")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	want := "You are an Expert Backend Engineer & Tech Lead\n\nUser Request:\nHello"
	if got != want {
		t.Fatalf("Render() = %q, want %q", got, want)
	}
}

func TestRenderRemovesEmptyGeminiRoleLine(t *testing.T) {
	t.Parallel()

	got, err := Render("You are an {{role}}\n\nUser Request:\n{{prompt}}", "Hello", "")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	want := "User Request:\nHello"
	if got != want {
		t.Fatalf("Render() = %q, want %q", got, want)
	}
}

func TestRenderRemovesEmptyGeminiRoleLineWithinDefaultShape(t *testing.T) {
	t.Parallel()

	got, err := Render("You are an {{role}}\n\nFollow these steps to answer:\n1. Think\n\nUser Request:\n{{prompt}}", "Hello", "")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	want := "Follow these steps to answer:\n1. Think\n\nUser Request:\nHello"
	if got != want {
		t.Fatalf("Render() = %q, want %q", got, want)
	}
}
