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

func TestRenderDataSubstitutesExtendedPlaceholders(t *testing.T) {
	t.Parallel()

	got, err := RenderData("Role: {{role}}\nTarget: {{target}}\nContext: {{context}}\nOutput Format: {{output_format}}\n\n{{prompt}}", Data{
		Prompt:       "Hello",
		Role:         "Backend reviewer",
		Target:       "claude",
		Context:      "service migration",
		OutputFormat: "bullets",
	})
	if err != nil {
		t.Fatalf("RenderData() error = %v", err)
	}

	want := "Role: Backend reviewer\nTarget: claude\nContext: service migration\nOutput Format: bullets\n\nHello"
	if got != want {
		t.Fatalf("RenderData() = %q, want %q", got, want)
	}
}

func TestRenderDataRemovesEmptyLabelLines(t *testing.T) {
	t.Parallel()

	got, err := RenderData("Role: {{role}}\nTarget: {{target}}\nContext: {{context}}\nOutput Format: {{output_format}}\n\n{{prompt}}", Data{
		Prompt: "Hello",
	})
	if err != nil {
		t.Fatalf("RenderData() error = %v", err)
	}

	want := "Hello"
	if got != want {
		t.Fatalf("RenderData() = %q, want %q", got, want)
	}
}

func TestRenderDataRemovesEmptyCodeLabelLines(t *testing.T) {
	t.Parallel()

	got, err := RenderData("// Target: {{target}}\n// Role: {{role}}\n// Context: {{context}}\n\n{{prompt}}", Data{
		Prompt: "Hello",
	})
	if err != nil {
		t.Fatalf("RenderData() error = %v", err)
	}

	want := "Hello"
	if got != want {
		t.Fatalf("RenderData() = %q, want %q", got, want)
	}
}

func TestRenderDataRemovesEmptyXMLBlocks(t *testing.T) {
	t.Parallel()

	got, err := RenderData("<role>\n{{role}}\n</role>\n<context>\n{{context}}\n</context>\n<input_prompt>\n{{prompt}}\n</input_prompt>", Data{
		Prompt: "Hello",
	})
	if err != nil {
		t.Fatalf("RenderData() error = %v", err)
	}

	want := "<input_prompt>\nHello\n</input_prompt>"
	if got != want {
		t.Fatalf("RenderData() = %q, want %q", got, want)
	}
}
