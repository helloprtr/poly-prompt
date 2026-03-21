package session

import (
	"strings"
)

// BuildStartPrompt constructs the initial prompt for a brand-new session.
func BuildStartPrompt(s Session) string {
	return buildPrompt(s, "", "", "시작해주세요.")
}

// BuildHandoffPrompt constructs the context-restoration prompt for model handoff.
// diff and lastResponse are optional — omit sections when empty.
func BuildHandoffPrompt(s Session, diff, lastResponse string) string {
	return buildPrompt(s, diff, lastResponse, "이어서 작업해주세요.")
}

func buildPrompt(s Session, diff, lastResponse, closing string) string {
	var b strings.Builder

	b.WriteString("[작업 목표]\n")
	b.WriteString(s.TaskGoal)
	if len(s.Constraints) > 0 {
		b.WriteString(" (" + strings.Join(s.Constraints, ", ") + ")")
	}
	b.WriteString("\n")

	if len(s.Files) > 0 {
		b.WriteString("\n[파일 범위]\n")
		b.WriteString(strings.Join(s.Files, "\n"))
		b.WriteString("\n")
	}

	if len(s.Checkpoints) > 0 {
		b.WriteString("\n[진행 상황]\n")
		for _, cp := range s.Checkpoints {
			b.WriteString("- " + cp.Note + "\n")
		}
	}

	if strings.TrimSpace(diff) != "" {
		b.WriteString("\n[코드 변화]\n")
		b.WriteString(diff)
		b.WriteString("\n")
	}

	if strings.TrimSpace(lastResponse) != "" {
		b.WriteString("\n[마지막 AI 응답]\n")
		b.WriteString(lastResponse)
		b.WriteString("\n")
	}

	b.WriteString("\n" + closing + "\n")
	return b.String()
}
