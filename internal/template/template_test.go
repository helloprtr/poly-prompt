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
