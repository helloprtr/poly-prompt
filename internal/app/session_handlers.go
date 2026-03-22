package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/helloprtr/poly-prompt/internal/config"
	"github.com/helloprtr/poly-prompt/internal/session"
)

// resolveCurrentSession finds the active session for the current git repo.
func (a *App) resolveCurrentSession() (session.Session, error) {
	root, err := a.resolveRepoRoot()
	if err != nil {
		return session.Session{}, session.ErrNoActiveSession
	}
	return a.sessionStore.ActiveFor(session.RepoHash(root))
}

func humanizeTime(t time.Time) string {
	d := time.Since(t)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return "방금 전"
	case d < time.Hour:
		return fmt.Sprintf("%d분 전", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d시간 전", int(d.Hours()))
	default:
		return fmt.Sprintf("%d일 전", int(d.Hours()/24))
	}
}

func (a *App) runCheckpoint(_ context.Context, note string) error {
	sess, err := a.resolveCurrentSession()
	if err != nil {
		return fmt.Errorf("no active session; run prtr review|edit|fix|design first")
	}
	root, _ := a.resolveRepoRoot()
	sha, _ := session.CurrentSHA(root)

	sess.Checkpoints = append(sess.Checkpoints, session.Checkpoint{
		Note:   note,
		GitSHA: sha,
		At:     time.Now().UTC(),
	})
	if err := a.sessionStore.Update(sess); err != nil {
		return fmt.Errorf("save checkpoint: %w", err)
	}
	fmt.Fprintf(a.stderr, "✓ 체크포인트 저장: %q\n", note)
	return nil
}

func (a *App) runDone(_ context.Context) error {
	sess, err := a.resolveCurrentSession()
	if err != nil {
		return fmt.Errorf("no active session")
	}
	if err := a.sessionStore.Complete(sess); err != nil {
		return fmt.Errorf("complete session: %w", err)
	}
	fmt.Fprintf(a.stderr, "✓ 세션 완료: %q\n", sess.TaskGoal)
	return nil
}

func (a *App) runSessions(_ context.Context) error {
	sessions, err := a.sessionStore.List()
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}

	// Filter to current repo when inside a git repo; show all otherwise.
	if root, err := a.resolveRepoRoot(); err == nil {
		hash := session.RepoHash(root)
		var filtered []session.Session
		for _, s := range sessions {
			if s.RepoHash == hash {
				filtered = append(filtered, s)
			}
		}
		sessions = filtered
	}

	if len(sessions) == 0 {
		fmt.Fprintln(a.stdout, "세션 없음.")
		return nil
	}
	for _, s := range sessions {
		status := "●"
		if s.Status == session.StatusCompleted {
			status = "✓"
		}
		fmt.Fprintf(a.stdout, "%s [%s] %q — %s (%s)\n",
			status, s.ID, s.TaskGoal, humanizeTime(s.LastActivity), s.TargetModel)
	}
	return nil
}

// readLastResponse reads $XDG_CONFIG_HOME/prtr/last-response.json (falling back to ~/.config)
// and returns the response field. Returns empty string on any error.
func (a *App) readLastResponse() string {
	base := strings.TrimSpace(a.lookupEnvOrDefault("XDG_CONFIG_HOME", ""))
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".config")
	}
	path := filepath.Join(base, "prtr", "last-response.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var v struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return ""
	}
	return v.Response
}

// captureSessionOnExit runs after the TUI exits, updating last_activity in the session.
// The git diff is NOT stored in the session — it is recomputed from base_git_sha at handoff time.
func (a *App) captureSessionOnExit(sess session.Session) error {
	sess.LastActivity = time.Now().UTC()
	if err := a.sessionStore.Update(sess); err != nil {
		return fmt.Errorf("update session after exit: %w", err)
	}
	fmt.Fprintln(a.stderr, "✓ 세션 저장됨 — 다음에 prtr로 이어서")
	return nil
}

// launchWithSession writes the start prompt to clipboard and launches the AI TUI as a subprocess.
func (a *App) launchWithSession(ctx context.Context, sess session.Session) error {
	cfg, err := a.configLoader()
	if err != nil {
		return err
	}

	prompt := session.BuildStartPrompt(sess)
	if err := a.clipboard.Copy(ctx, prompt); err != nil {
		return fmt.Errorf("copy session prompt: %w", err)
	}

	model := sess.TargetModel
	if model == "" {
		model = cfg.DefaultTarget
	}
	bins := session.ModelBinaries(model)
	binary, err := session.FindBinary(bins...)
	if err != nil {
		fmt.Fprintf(a.stderr, "✓ 프롬프트가 클립보드에 복사됐습니다 (%s를 찾을 수 없어 직접 열어주세요)\n", model)
		return nil
	}

	fmt.Fprintln(a.stderr, "✓ 세션 시작 — 프롬프트가 클립보드에 복사됐습니다. TUI에 붙여넣으세요.")
	if err := session.RunForeground(ctx, binary); err != nil {
		fmt.Fprintf(a.stderr, "AI 프로세스 종료: %v\n", err)
	}
	return a.captureSessionOnExit(sess)
}

// launchHandoff builds the handoff prompt and launches the target model.
// resolveRepoRoot errors are silently ignored — diff is best-effort.
func (a *App) launchHandoff(ctx context.Context, sess session.Session, model string) error {
	root, _ := a.resolveRepoRoot()
	diff, _ := session.Diff(root, sess.BaseGitSHA)
	lastResp := a.readLastResponse()

	prompt := session.BuildHandoffPrompt(sess, diff, lastResp)
	if err := a.clipboard.Copy(ctx, prompt); err != nil {
		return fmt.Errorf("copy handoff prompt: %w", err)
	}

	bins := session.ModelBinaries(model)
	binary, err := session.FindBinary(bins...)
	if err != nil {
		fmt.Fprintf(a.stderr, "✓ 핸드오프 프롬프트가 클립보드에 복사됐습니다 (%s를 찾을 수 없어 직접 열어주세요)\n", model)
		return nil
	}

	fmt.Fprintf(a.stderr, "✓ %s로 핸드오프 — 프롬프트가 클립보드에 복사됐습니다.\n", model)
	if err := session.RunForeground(ctx, binary); err != nil {
		fmt.Fprintf(a.stderr, "AI 프로세스 종료: %v\n", err)
	}
	sess.TargetModel = model
	return a.captureSessionOnExit(sess)
}

// runSessionCreate runs the interactive new-session creation flow.
// mode and files may be pre-populated from command args.
// reader must be a *bufio.Reader wrapping the stdin io.Reader.
func (a *App) runSessionCreate(ctx context.Context, mode session.Mode, files []string, reader *bufio.Reader) error {
	fmt.Fprint(a.stderr, "무엇을 하려 하나요? ")
	goal, _ := reader.ReadString('\n')
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return fmt.Errorf("작업 목표를 입력해주세요")
	}

	if len(files) == 0 {
		fmt.Fprint(a.stderr, "어떤 파일? (Enter로 건너뜀) ")
		line, _ := reader.ReadString('\n')
		if line = strings.TrimSpace(line); line != "" {
			files = strings.Fields(line)
		}
	}

	fmt.Fprint(a.stderr, "제약조건? (Enter로 건너뜀) ")
	constraintLine, _ := reader.ReadString('\n')
	var constraints []string
	if cl := strings.TrimSpace(constraintLine); cl != "" {
		for _, c := range strings.Split(cl, ",") {
			if t := strings.TrimSpace(c); t != "" {
				constraints = append(constraints, t)
			}
		}
	}

	root, err := a.resolveRepoRoot()
	if err != nil {
		return fmt.Errorf("prtr sessions require a git repository: %w", err)
	}
	hash := session.RepoHash(root)
	sha, _ := session.CurrentSHA(root)

	cfg, err := a.configLoader()
	if err != nil {
		cfg = config.Config{}
	}
	sess := session.Session{
		Repo:        root,
		RepoHash:    hash,
		TaskGoal:    goal,
		Files:       files,
		Mode:        mode,
		Constraints: constraints,
		TargetModel: cfg.DefaultTarget,
		Status:      session.StatusActive,
		StartedAt:   time.Now().UTC(),
		BaseGitSHA:  sha,
	}

	if err := a.sessionStore.Save(sess); err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	return a.launchWithSession(ctx, sess)
}

// runBare implements the bare `prtr` command (no subcommand).
func (a *App) runBare(ctx context.Context, stdin io.Reader) error {
	sess, err := a.resolveCurrentSession()
	if err == nil {
		return a.offerContinueOrNew(ctx, sess, stdin)
	}
	return a.runSessionCreate(ctx, session.ModeEdit, nil, bufio.NewReader(stdin))
}

func (a *App) offerContinueOrNew(ctx context.Context, sess session.Session, stdin io.Reader) error {
	fmt.Fprintf(a.stderr, "─────────────────────────────────────\n")
	fmt.Fprintf(a.stderr, "이어서 할까요?\n\n")
	fmt.Fprintf(a.stderr, "%q — %s (%s)\n", sess.TaskGoal, humanizeTime(sess.LastActivity), sess.TargetModel)
	fmt.Fprintf(a.stderr, "─────────────────────────────────────\n")
	fmt.Fprint(a.stderr, "[Enter] 이어서  [n] 새 작업: ")

	reader := bufio.NewReader(stdin)
	line, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(line)) == "n" {
		return a.runSessionCreate(ctx, session.ModeEdit, nil, reader)
	}
	return a.launchHandoff(ctx, sess, sess.TargetModel)
}

// runSessionMode handles prtr review|edit|fix|design [files...]
func (a *App) runSessionMode(ctx context.Context, mode session.Mode, args []string, stdin io.Reader) error {
	reader := bufio.NewReader(stdin)
	if sess, err := a.resolveCurrentSession(); err == nil {
		fmt.Fprintf(a.stderr, "⚡ 진행 중인 세션: %q (%s)\n", sess.TaskGoal, humanizeTime(sess.LastActivity))
		fmt.Fprint(a.stderr, "이어서 할까요, 새로 시작할까요? [이어서/새로]: ")
		line, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(line)) != "새로" {
			return a.launchHandoff(ctx, sess, sess.TargetModel)
		}
	}
	return a.runSessionCreate(ctx, mode, args, reader)
}

// runHandoff handles `prtr @gemini` / `prtr @codex`.
func (a *App) runHandoff(ctx context.Context, model string) error {
	sess, err := a.resolveCurrentSession()
	if err != nil {
		return fmt.Errorf("no active session; run prtr review|edit|fix|design first")
	}
	return a.launchHandoff(ctx, sess, model)
}

// runCapsuleStatus is a stub for Work Capsule drift info.
// TODO(v1.1): integrate Work Capsule drift reporting here.
func (a *App) runCapsuleStatus(_ context.Context) error { return nil }

func (a *App) runStatus(ctx context.Context) error {
	sess, err := a.resolveCurrentSession()
	if err == nil {
		root, _ := a.resolveRepoRoot()
		diff, _ := session.Diff(root, sess.BaseGitSHA)

		fmt.Fprintln(a.stdout, "[현재 세션]")
		fmt.Fprintf(a.stdout, "작업: %s\n", sess.TaskGoal)
		if len(sess.Files) > 0 {
			fmt.Fprintf(a.stdout, "파일: %s\n", strings.Join(sess.Files, ", "))
		}
		fmt.Fprintf(a.stdout, "모드: %s\n", sess.Mode)
		fmt.Fprintf(a.stdout, "시작: %s (%s)\n", humanizeTime(sess.StartedAt), sess.TargetModel)
		if summary := summarizeDiff(diff); summary != "" {
			fmt.Fprintf(a.stdout, "변경: %s\n", summary)
		}
		if len(sess.Checkpoints) > 0 {
			last := sess.Checkpoints[len(sess.Checkpoints)-1]
			fmt.Fprintf(a.stdout, "체크포인트: %q\n", last.Note)
		}
		fmt.Fprintln(a.stdout)
	} else {
		fmt.Fprintf(a.stdout, "[현재 세션]\n세션 없음\n\n")
	}

	return a.runCapsuleStatus(ctx)
}

func summarizeDiff(diff string) string {
	if diff == "" {
		return ""
	}
	var files []string
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "diff --git") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				files = append(files, strings.TrimPrefix(parts[3], "b/"))
			}
		}
	}
	if len(files) == 0 {
		return ""
	}
	if len(files) > 3 {
		return fmt.Sprintf("%s 외 %d개", strings.Join(files[:3], ", "), len(files)-3)
	}
	return strings.Join(files, ", ")
}
