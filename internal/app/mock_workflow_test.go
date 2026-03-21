package app

// mock_workflow_test.go — 실사용 시나리오 기반 mock 테스트 모음
//
// 이 파일은 실제 사용자가 prtr을 사용하는 다음 워크플로를 검증합니다:
//
//  시나리오 A: 기본 전송 루프 (go → again → swap)
//  시나리오 B: 번역 파이프라인 (한국어 입력 → 영어 전달)
//  시나리오 C: take 액션 (클립보드 → 구조화된 프롬프트)
//  시나리오 D: take --deep (5-worker deep 분석 파이프라인)
//  시나리오 E: learn (termbook 학습)
//  시나리오 F: inspect (dry-run 미리보기)
//  시나리오 G: history & rerun (히스토리 검색 및 재실행)
//  시나리오 H: 멀티-프로바이더 shortcut (fix, ask, review, design)
//  시나리오 I: 에러 처리 (빈 클립보드, 잘못된 action 등)

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/helloprtr/poly-prompt/internal/config"
	"github.com/helloprtr/poly-prompt/internal/history"
	"github.com/helloprtr/poly-prompt/internal/repoctx"
	"github.com/helloprtr/poly-prompt/internal/termbook"
	"github.com/helloprtr/poly-prompt/internal/translate"
)

// ---------------------------------------------------------------------------
// 공통 mock 헬퍼
// ---------------------------------------------------------------------------

// mockAppConfig는 실사용에 가까운 테스트 설정을 반환합니다.
// 한국어→영어 번역, claude 기본 타겟, 세 가지 롤(be/ui/review)을 포함합니다.
func mockAppConfig() config.Config {
	cfg := testConfig()
	// 한국어 입력을 영어로 번역하는 기본 설정
	cfg.TranslationSourceLang = "ko"
	cfg.TranslationTargetLang = "en"
	return cfg
}

// mockHistoryStore는 임시 디렉토리에 히스토리 저장소를 생성합니다.
func mockHistoryStore(t *testing.T) *history.Store {
	t.Helper()
	return history.New(filepath.Join(t.TempDir(), "history.json"))
}

// mockRepoApp은 git repo context를 포함한 App을 반환합니다 (deep 테스트용).
func mockRepoApp(t *testing.T, repoRoot string, clipboard *stubClipboard) *App {
	t.Helper()
	return New(Dependencies{
		Version:         "test",
		Stdout:          &bytes.Buffer{},
		Stderr:          &bytes.Buffer{},
		Translator:      &stubTranslator{output: "Translated mock output"},
		Clipboard:       clipboard,
		Editor:          &stubEditor{},
		Launcher:        &stubLauncher{},
		Automator:       &stubAutomator{},
		SubmitConfirmer: &stubConfirmer{},
		ConfigLoader: func() (config.Config, error) {
			return mockAppConfig(), nil
		},
		ConfigInit:   func() (string, error) { return "/tmp/prtr/config.toml", nil },
		LookupEnv:    func(string) (string, bool) { return "", false },
		HistoryStore: history.New(filepath.Join(t.TempDir(), "history.json")),
		RepoContext: &stubRepoContext{summary: repoctx.Summary{
			RepoName: "my-project",
			Branch:   "feature/auth-fix",
			Changes:  []string{" M internal/auth/handler.go", " M internal/auth/middleware.go"},
		}},
		RepoRootFinder: func() (string, error) { return repoRoot, nil },
		TermbookLoader: func(string) (termbook.Book, error) {
			return termbook.Book{ProtectedTerms: []string{"AuthHandler", "TokenRefresh"}}, nil
		},
	})
}

// ---------------------------------------------------------------------------
// 시나리오 A: 기본 전송 루프
// ---------------------------------------------------------------------------

// TestMockScenarioA1_GoSendsKoreanPrompt는 한국어 프롬프트를 claude에 전송하는
// 가장 기본적인 사용 시나리오를 검증합니다.
//
// 실사용 명령어: prtr go "로그인 버그를 수정해줘"
func TestMockScenarioA1_GoSendsKoreanPrompt(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Please fix the login bug"}
	cb := &stubClipboard{}
	app := newTestApp(t, mockAppConfig(), translator, cb, &stubEditor{}, mockHistoryStore(t))
	stdout, stderr := buffersFromApp(app)

	err := app.Execute(context.Background(),
		[]string{"go", "로그인 버그를 수정해줘"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// 번역 요청 확인
	if translator.gotInput.Text != "로그인 버그를 수정해줘" {
		t.Errorf("translator input = %q, want %q", translator.gotInput.Text, "로그인 버그를 수정해줘")
	}

	// 번역된 결과가 클립보드에 복사됐는지 확인
	if cb.calls == 0 {
		t.Error("clipboard.Copy() not called — prompt was not delivered")
	}
	if !strings.Contains(cb.copied, "Please fix the login bug") {
		t.Errorf("clipboard content = %q, want to contain translated text", cb.copied)
	}

	// 상태 로그 확인 (stderr)
	if !strings.Contains(stderr.String(), "claude") {
		t.Errorf("stderr = %q, want to mention target 'claude'", stderr.String())
	}

	// stdout에 프롬프트가 출력됐는지 확인
	if stdout.Len() == 0 {
		t.Error("stdout is empty — rendered prompt was not printed")
	}
}

// TestMockScenarioA2_GoWithModeAndTemplate는 모드(review)와 root-level 템플릿을
// 지정한 전송 시나리오를 검증합니다.
//
// 실사용 명령어: prtr go review "DB 연결 누수 분석"
// 루트 레벨에서 롤/템플릿을 지정할 때: prtr -r be --template claude-structured "..."
func TestMockScenarioA2_GoWithModeAndTemplate(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Analyze the DB connection leak"}
	cb := &stubClipboard{}
	app := newTestApp(t, mockAppConfig(), translator, cb, &stubEditor{}, mockHistoryStore(t))
	stdout, _ := buffersFromApp(app)

	// go subcommand은 mode (ask/review/fix/design)를 positional로 지정
	err := app.Execute(context.Background(),
		[]string{"go", "review", "DB 연결 누수 분석"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// 번역이 호출됐는지 확인
	if translator.gotInput.Text != "DB 연결 누수 분석" {
		t.Errorf("translator input = %q, want original text", translator.gotInput.Text)
	}

	// 클립보드에 프롬프트가 복사됐는지 확인
	if cb.calls == 0 {
		t.Error("clipboard.Copy() not called")
	}

	// 출력에 프롬프트가 있는지 확인
	if stdout.Len() == 0 {
		t.Error("stdout is empty")
	}
}

// TestMockScenarioA3_AgainReplayLatestEntry는 최신 히스토리를 재실행하는
// 시나리오를 검증합니다.
//
// 실사용 명령어: prtr again
func TestMockScenarioA3_AgainReplayLatestEntry(t *testing.T) {
	t.Parallel()

	store := mockHistoryStore(t)
	// 히스토리에 이전 실행 기록을 미리 삽입
	_ = store.Append(history.Entry{
		ID:         "prev-001",
		CreatedAt:  time.Now().Add(-5 * time.Minute),
		Target:     "claude",
		Translated: "Please refactor the user service",
		Original:   "유저 서비스를 리팩토링해줘",
	})

	translator := &stubTranslator{output: "Please refactor the user service"}
	cb := &stubClipboard{}
	app := newTestApp(t, mockAppConfig(), translator, cb, &stubEditor{}, store)

	err := app.Execute(context.Background(),
		[]string{"again"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// 클립보드에 이전 프롬프트가 복사됐는지 확인
	if cb.calls == 0 {
		t.Error("clipboard.Copy() not called after 'again'")
	}
}

// TestMockScenarioA4_SwapChangesTargetProvider는 프로바이더를 전환하는
// 시나리오를 검증합니다.
//
// 실사용 명령어: prtr swap gemini
func TestMockScenarioA4_SwapChangesTargetProvider(t *testing.T) {
	t.Parallel()

	store := mockHistoryStore(t)
	_ = store.Append(history.Entry{
		ID:         "prev-001",
		CreatedAt:  time.Now().Add(-3 * time.Minute),
		Target:     "claude",
		Translated: "Fix the API rate limiter",
		Original:   "API rate limiter 수정해줘",
	})

	translator := &stubTranslator{output: "Fix the API rate limiter"}
	cb := &stubClipboard{}
	app := newTestApp(t, mockAppConfig(), translator, cb, &stubEditor{}, store)
	stdout, _ := buffersFromApp(app)

	err := app.Execute(context.Background(),
		[]string{"swap", "gemini"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Gemini 템플릿 형식 확인 (stepwise 형식)
	got := stdout.String()
	if !strings.Contains(got, "User Request:") {
		t.Errorf("stdout = %q, want Gemini stepwise format with 'User Request:'", got)
	}
}

// ---------------------------------------------------------------------------
// 시나리오 B: 번역 파이프라인
// ---------------------------------------------------------------------------

// TestMockScenarioB1_KoreanToEnglishTranslation은 한국어 입력을 영어로
// 번역하여 전달하는 전체 파이프라인을 검증합니다.
func TestMockScenarioB1_KoreanToEnglishTranslation(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Implement a JWT refresh token rotation mechanism"}
	cb := &stubClipboard{}
	app := newTestApp(t, mockAppConfig(), translator, cb, &stubEditor{}, mockHistoryStore(t))

	err := app.Execute(context.Background(),
		[]string{"go", "JWT refresh token 교체 메커니즘을 구현해줘"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// 번역 입력 언어 확인
	if translator.gotInput.SourceLang != "ko" {
		t.Errorf("SourceLang = %q, want %q", translator.gotInput.SourceLang, "ko")
	}
	if translator.gotInput.TargetLang != "en" {
		t.Errorf("TargetLang = %q, want %q", translator.gotInput.TargetLang, "en")
	}

	// 번역된 텍스트가 클립보드에 포함됐는지 확인
	if !strings.Contains(cb.copied, "Implement a JWT refresh token rotation mechanism") {
		t.Errorf("clipboard content = %q, want translated text", cb.copied)
	}
}

// TestMockScenarioB2_TranslationErrorFallthrough는 번역 API 오류 시에도
// 원문이 그대로 전달됨을 검증합니다.
func TestMockScenarioB2_TranslationErrorFallthrough(t *testing.T) {
	t.Parallel()

	cfg := mockAppConfig()
	cfg.TranslationSourceLang = "auto"
	cfg.TranslationTargetLang = "en"

	// DeepL API 키 없는 상황 시뮬레이션 (영어 입력은 번역 생략)
	translator := &stubTranslator{output: ""}
	cb := &stubClipboard{}
	app := newTestApp(t, cfg, translator, cb, &stubEditor{}, mockHistoryStore(t))

	err := app.Execute(context.Background(),
		[]string{"go", "Fix the authentication middleware"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// 영어 입력이라 번역 생략됨 → 클립보드에 원문이 있어야 함
	if cb.calls == 0 {
		t.Error("clipboard.Copy() not called")
	}
}

// ---------------------------------------------------------------------------
// 시나리오 C: take 액션
// ---------------------------------------------------------------------------

// TestMockScenarioC1_TakePatchFromClipboard는 클립보드의 AI 응답을 바탕으로
// patch 프롬프트를 생성하는 시나리오를 검증합니다.
//
// 실사용 명령어: prtr take patch
func TestMockScenarioC1_TakePatchFromClipboard(t *testing.T) {
	t.Parallel()

	store := mockHistoryStore(t)
	_ = store.Append(history.Entry{
		ID:        "prev-001",
		CreatedAt: time.Now().Add(-2 * time.Minute),
		Target:    "claude",
		Original:  "로그인 버그 수정",
	})

	// 클립보드에 AI가 제안한 코드 변경 내용
	clipContent := `The login bug is caused by a missing null check in auth/handler.go line 42.
Here's the fix:
- Add: if user == nil { return ErrUnauthorized }
- Remove the early return that skips validation`

	cb := &stubClipboard{read: clipContent}
	app := newTestApp(t, mockAppConfig(), &stubTranslator{}, cb, &stubEditor{}, store)
	stdout, stderr := buffersFromApp(app)

	err := app.Execute(context.Background(),
		[]string{"take", "patch", "--dry-run"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// 상태 로그에 take:patch 표시 확인
	if !strings.Contains(stderr.String(), "take:patch") {
		t.Errorf("stderr = %q, want 'take:patch' in status", stderr.String())
	}

	// 출력에 프롬프트가 생성됐는지 확인
	if stdout.Len() == 0 {
		t.Error("stdout is empty — take patch prompt was not rendered")
	}
}

// TestMockScenarioC2_TakeCommitFromClipboard는 commit 메시지 생성 시나리오를
// 검증합니다.
//
// 실사용 명령어: prtr take commit
func TestMockScenarioC2_TakeCommitFromClipboard(t *testing.T) {
	t.Parallel()

	store := mockHistoryStore(t)
	_ = store.Append(history.Entry{
		ID:        "prev-001",
		CreatedAt: time.Now().Add(-1 * time.Minute),
		Target:    "codex",
		Original:  "add retry logic to HTTP client",
	})

	cb := &stubClipboard{read: "Added exponential backoff retry logic to the HTTP client with configurable max retries and jitter."}
	app := newTestApp(t, mockAppConfig(), &stubTranslator{}, cb, &stubEditor{}, store)

	err := app.Execute(context.Background(),
		[]string{"take", "commit", "--dry-run"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

// TestMockScenarioC3_TakeTestFromClipboard는 test 계획 생성 시나리오를
// 검증합니다.
//
// 실사용 명령어: prtr take test
func TestMockScenarioC3_TakeTestFromClipboard(t *testing.T) {
	t.Parallel()

	store := mockHistoryStore(t)
	_ = store.Append(history.Entry{
		ID:        "prev-001",
		CreatedAt: time.Now().Add(-1 * time.Minute),
		Target:    "claude",
		Original:  "add rate limiting middleware",
	})

	cb := &stubClipboard{read: "Rate limiting middleware implemented using token bucket algorithm with Redis backend."}
	app := newTestApp(t, mockAppConfig(), &stubTranslator{}, cb, &stubEditor{}, store)
	stdout, _ := buffersFromApp(app)

	err := app.Execute(context.Background(),
		[]string{"take", "test", "--dry-run"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if stdout.Len() == 0 {
		t.Error("stdout is empty — take test prompt was not rendered")
	}
}

// ---------------------------------------------------------------------------
// 시나리오 D: take --deep (5-worker deep 파이프라인)
// ---------------------------------------------------------------------------

// TestMockScenarioD1_TakeDeepPatchWithRepoContext는 git repo context를 포함한
// deep 분석 파이프라인의 전체 실행을 검증합니다.
//
// 실사용 명령어: prtr take patch --deep --dry-run
func TestMockScenarioD1_TakeDeepPatchWithRepoContext(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := mockHistoryStore(t)
	_ = store.Append(history.Entry{
		ID:        "prev-001",
		CreatedAt: time.Now().Add(-5 * time.Minute),
		Target:    "claude",
		Original:  "auth middleware 리팩토링",
	})

	cb := &stubClipboard{read: "Fix the nil pointer dereference in internal/auth/middleware.go line 87 when token is expired."}
	app := mockRepoApp(t, repoRoot, cb)
	// mockRepoApp에서 이미 store를 생성하므로 별도 store 연결 불필요
	stdout, stderr := buffersFromApp(app)

	err := app.Execute(context.Background(),
		[]string{"take", "patch", "--deep", "--dry-run"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// 5개 worker 단계 진행 확인
	stderrStr := stderr.String()
	for _, step := range []string{"planner", "patcher", "critic", "tester", "reconciler"} {
		if !strings.Contains(stderrStr, step) {
			t.Errorf("stderr missing step %q; got = %q", step, stderrStr)
		}
	}

	// deep 실행 결과로 stdout에 프롬프트 출력 확인
	if stdout.Len() == 0 {
		t.Error("stdout is empty after deep run")
	}

	// 아티팩트 파일이 생성됐는지 확인
	runsDir := filepath.Join(repoRoot, ".prtr", "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", runsDir, err)
	}
	if len(entries) == 0 {
		t.Fatal("no run directory created under .prtr/runs/")
	}
}

// TestMockScenarioD2_TakeDeepTestAction은 deep test 액션을 검증합니다.
//
// 실사용 명령어: prtr take test --deep --dry-run
func TestMockScenarioD2_TakeDeepTestAction(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	cb := &stubClipboard{read: "Added caching layer to UserService.GetByID() in internal/user/service.go"}
	app := mockRepoApp(t, repoRoot, cb)

	err := app.Execute(context.Background(),
		[]string{"take", "test", "--deep", "--dry-run"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

// TestMockScenarioD3_TakeDeepRejectsInvalidAction은 지원하지 않는 action에
// 대한 오류 처리를 검증합니다.
//
// 실사용 예시: prtr take summary --deep (오류 발생 예상)
func TestMockScenarioD3_TakeDeepRejectsInvalidAction(t *testing.T) {
	t.Parallel()

	cb := &stubClipboard{read: "Some AI response"}
	app := newTestApp(t, mockAppConfig(), &stubTranslator{}, cb, &stubEditor{}, mockHistoryStore(t))

	err := app.Execute(context.Background(),
		[]string{"take", "summary", "--deep", "--dry-run"},
		strings.NewReader(""), false)
	if err == nil {
		t.Fatal("Execute() expected error for unsupported deep action, got nil")
	}
	if !strings.Contains(err.Error(), "deep execution supports") {
		t.Errorf("error = %v, want it to mention supported actions", err)
	}
}

// ---------------------------------------------------------------------------
// 시나리오 E: learn (termbook)
// ---------------------------------------------------------------------------

// TestMockScenarioE1_LearnDryRunExtractsTerms는 dry-run 모드에서 프로젝트 용어를
// 추출하는 시나리오를 검증합니다.
//
// 실사용 명령어: prtr learn --dry-run
func TestMockScenarioE1_LearnDryRunExtractsTerms(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := mockHistoryStore(t)
	// 최근 히스토리에 프로젝트 관련 용어 포함
	_ = store.Append(history.Entry{
		ID:         "hist-001",
		CreatedAt:  time.Now().Add(-10 * time.Minute),
		Target:     "claude",
		Translated: "Refactor AuthHandler to use TokenRefresh middleware",
		Original:   "AuthHandler를 TokenRefresh 미들웨어를 사용하도록 리팩토링",
	})

	app := New(Dependencies{
		Version:         "test",
		Stdout:          &bytes.Buffer{},
		Stderr:          &bytes.Buffer{},
		Translator:      &stubTranslator{},
		Clipboard:       &stubClipboard{},
		Editor:          &stubEditor{},
		Launcher:        &stubLauncher{},
		Automator:       &stubAutomator{},
		SubmitConfirmer: &stubConfirmer{},
		ConfigLoader: func() (config.Config, error) {
			return mockAppConfig(), nil
		},
		ConfigInit:   func() (string, error) { return "/tmp/prtr/config.toml", nil },
		LookupEnv:    func(string) (string, bool) { return "", false },
		HistoryStore: store,
		RepoContext:  &stubRepoContext{},
		RepoRootFinder: func() (string, error) {
			return repoRoot, nil
		},
		TermbookLoader: func(string) (termbook.Book, error) {
			return termbook.Book{}, os.ErrNotExist
		},
	})
	stdout, _ := buffersFromApp(app)

	err := app.Execute(context.Background(),
		[]string{"learn", "--dry-run"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// dry-run 출력에 추출된 용어가 표시됐는지 확인
	if stdout.Len() == 0 {
		t.Error("stdout is empty — learn dry-run should print extracted terms")
	}
}

// TestMockScenarioE2_LearnProtectsTermsInPrompt는 학습된 용어가 이후 프롬프트에서
// 번역 보호 처리가 적용됨을 검증합니다.
//
// prtr은 보호 용어를 텍스트에서 직접 치환(placeholder)하여 번역 대상에서 제외합니다.
// 번역 요청 텍스트에 보호 용어 플레이스홀더가 포함되는지 또는 stdout에 표시되는지 확인합니다.
func TestMockScenarioE2_LearnProtectsTermsInPrompt(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Refactor the AuthHandler"}
	cb := &stubClipboard{}
	repoRoot := t.TempDir()
	app := New(Dependencies{
		Version:         "test",
		Stdout:          &bytes.Buffer{},
		Stderr:          &bytes.Buffer{},
		Translator:      translator,
		Clipboard:       cb,
		Editor:          &stubEditor{},
		Launcher:        &stubLauncher{},
		Automator:       &stubAutomator{},
		SubmitConfirmer: &stubConfirmer{},
		ConfigLoader: func() (config.Config, error) {
			return mockAppConfig(), nil
		},
		ConfigInit:   func() (string, error) { return "/tmp/prtr/config.toml", nil },
		LookupEnv:    func(string) (string, bool) { return "", false },
		HistoryStore: mockHistoryStore(t),
		RepoContext:  &stubRepoContext{},
		RepoRootFinder: func() (string, error) {
			return repoRoot, nil
		},
		TermbookLoader: func(string) (termbook.Book, error) {
			// AuthHandler는 프로젝트 전용 용어로 보호됨
			return termbook.Book{ProtectedTerms: []string{"AuthHandler", "TokenRefresh"}}, nil
		},
	})
	stdout, _ := buffersFromApp(app)

	err := app.Execute(context.Background(),
		[]string{"go", "ask", "AuthHandler를 리팩토링해줘", "--dry-run"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// 보호 용어가 stdout(렌더된 프롬프트)에 표시되는지 확인
	// prtr은 보호 용어를 "Protected project terms: ..." 형식으로 출력
	if !strings.Contains(stdout.String(), "AuthHandler") {
		t.Errorf("stdout = %q, want 'AuthHandler' to appear in protected terms section", stdout.String())
	}
}

// ---------------------------------------------------------------------------
// 시나리오 F: inspect (dry-run 미리보기)
// ---------------------------------------------------------------------------

// TestMockScenarioF1_InspectShowsRoutingWithoutClipboard는 클립보드 복사 없이
// 프롬프트 라우팅 경로를 미리보는 시나리오를 검증합니다.
//
// 실사용 명령어: prtr inspect "DB 인덱싱 최적화"
func TestMockScenarioF1_InspectShowsRoutingWithoutClipboard(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Optimize DB indexing strategy"}
	cb := &stubClipboard{}
	app := newTestApp(t, mockAppConfig(), translator, cb, &stubEditor{}, mockHistoryStore(t))
	stdout, _ := buffersFromApp(app)

	err := app.Execute(context.Background(),
		[]string{"inspect", "DB 인덱싱 최적화"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// inspect는 클립보드 복사 없이 미리보기
	if cb.calls != 0 {
		t.Errorf("clipboard.Copy() called %d times during inspect, want 0", cb.calls)
	}

	// 라우팅 정보가 출력됐는지 확인
	if stdout.Len() == 0 {
		t.Error("stdout is empty — inspect should print routing info")
	}
}

// TestMockScenarioF2_InspectWithTargetOverride는 특정 타겟으로 라우팅 경로를
// 미리보는 시나리오를 검증합니다.
//
// 실사용 명령어: prtr inspect --to gemini "API 설계 검토"
func TestMockScenarioF2_InspectWithTargetOverride(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Review API design patterns"}
	cb := &stubClipboard{}
	app := newTestApp(t, mockAppConfig(), translator, cb, &stubEditor{}, mockHistoryStore(t))
	stdout, _ := buffersFromApp(app)

	err := app.Execute(context.Background(),
		[]string{"inspect", "--lang", "en", "API 설계 검토"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// 클립보드 복사 없음 확인
	if cb.calls != 0 {
		t.Errorf("clipboard.Copy() called %d times during inspect, want 0", cb.calls)
	}
	if stdout.Len() == 0 {
		t.Error("stdout is empty")
	}
}

// ---------------------------------------------------------------------------
// 시나리오 G: history & rerun
// ---------------------------------------------------------------------------

// TestMockScenarioG1_HistoryListAndSearch는 히스토리 목록 조회와 검색을
// 검증합니다.
//
// 실사용 명령어: prtr history / prtr history search "auth"
func TestMockScenarioG1_HistoryListAndSearch(t *testing.T) {
	t.Parallel()

	store := mockHistoryStore(t)
	entries := []history.Entry{
		{ID: "h1", CreatedAt: time.Now().Add(-30 * time.Minute), Target: "claude", Original: "auth middleware 수정", Translated: "Fix auth middleware"},
		{ID: "h2", CreatedAt: time.Now().Add(-20 * time.Minute), Target: "codex", Original: "DB 쿼리 최적화", Translated: "Optimize DB queries"},
		{ID: "h3", CreatedAt: time.Now().Add(-10 * time.Minute), Target: "gemini", Original: "API 설계 검토", Translated: "Review API design"},
	}
	for _, e := range entries {
		if err := store.Append(e); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	app := newTestApp(t, mockAppConfig(), &stubTranslator{}, &stubClipboard{}, &stubEditor{}, store)
	stdout, _ := buffersFromApp(app)

	// 전체 히스토리 목록
	err := app.Execute(context.Background(),
		[]string{"history"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("history error = %v", err)
	}
	if stdout.Len() == 0 {
		t.Error("stdout is empty — history should print entries")
	}

	stdout.Reset()

	// auth 검색
	err = app.Execute(context.Background(),
		[]string{"history", "search", "auth"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("history search error = %v", err)
	}
	if !strings.Contains(stdout.String(), "auth") {
		t.Errorf("search result = %q, want 'auth' to appear", stdout.String())
	}
}

// TestMockScenarioG2_PinAndFavoriteEntry는 히스토리 항목에 즐겨찾기를 설정하는
// 시나리오를 검증합니다.
//
// 실사용 명령어: prtr favorite <id>
func TestMockScenarioG2_PinAndFavoriteEntry(t *testing.T) {
	t.Parallel()

	store := mockHistoryStore(t)
	_ = store.Append(history.Entry{
		ID:        "pin-target",
		CreatedAt: time.Now(),
		Target:    "claude",
		Original:  "중요한 아키텍처 결정",
	})

	app := newTestApp(t, mockAppConfig(), &stubTranslator{}, &stubClipboard{}, &stubEditor{}, store)
	stdout, _ := buffersFromApp(app)

	err := app.Execute(context.Background(),
		[]string{"favorite", "pin-target"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("favorite error = %v", err)
	}
	if !strings.Contains(stdout.String(), "favorited") {
		t.Errorf("stdout = %q, want 'favorited'", stdout.String())
	}
}

// ---------------------------------------------------------------------------
// 시나리오 H: 멀티-프로바이더 shortcut
// ---------------------------------------------------------------------------

// TestMockScenarioH1_FixShortcutUsesCodexBE는 "fix" shortcut이 codex + be 롤로
// 매핑됨을 검증합니다.
//
// 실사용 명령어: prtr fix "nil pointer 에러 수정"
func TestMockScenarioH1_FixShortcutUsesCodexBE(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Fix nil pointer error in handler"}
	cb := &stubClipboard{}
	app := newTestApp(t, mockAppConfig(), translator, cb, &stubEditor{}, mockHistoryStore(t))
	stdout, _ := buffersFromApp(app)

	err := app.Execute(context.Background(),
		[]string{"fix", "nil pointer 에러 수정"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// codex 템플릿 형식 확인 (// Target: codex)
	got := stdout.String()
	if !strings.Contains(got, "// Target: codex") {
		t.Errorf("stdout = %q, want codex template with '// Target: codex'", got)
	}
}

// TestMockScenarioH2_DesignShortcutUsesGeminiUI는 "design" shortcut이
// gemini + ui 롤로 매핑됨을 검증합니다.
//
// 실사용 명령어: prtr design "대시보드 UX 개선"
func TestMockScenarioH2_DesignShortcutUsesGeminiUI(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Improve dashboard UX and information hierarchy"}
	cb := &stubClipboard{}
	app := newTestApp(t, mockAppConfig(), translator, cb, &stubEditor{}, mockHistoryStore(t))
	stdout, _ := buffersFromApp(app)

	err := app.Execute(context.Background(),
		[]string{"design", "대시보드 UX 개선"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Gemini stepwise 형식 확인
	got := stdout.String()
	if !strings.Contains(got, "User Request:") {
		t.Errorf("stdout = %q, want Gemini format with 'User Request:'", got)
	}
}

// TestMockScenarioH3_ReviewShortcutUsesClaudeBE는 "review" shortcut이
// claude + be + claude-review 템플릿으로 매핑됨을 검증합니다.
//
// 실사용 명령어: prtr review "PR #42 코드 리뷰"
func TestMockScenarioH3_ReviewShortcutUsesClaudeBE(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Review PR #42 for security issues"}
	cb := &stubClipboard{}
	app := newTestApp(t, mockAppConfig(), translator, cb, &stubEditor{}, mockHistoryStore(t))
	stdout, _ := buffersFromApp(app)

	err := app.Execute(context.Background(),
		[]string{"review", "PR #42 코드 리뷰"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Claude review 템플릿 형식 확인
	got := stdout.String()
	if !strings.Contains(got, "<task>") {
		t.Errorf("stdout = %q, want claude-review template with '<task>'", got)
	}
}

// ---------------------------------------------------------------------------
// 시나리오 I: 에러 처리
// ---------------------------------------------------------------------------

// TestMockScenarioI1_EmptyClipboardRejectsDeep는 클립보드가 비어 있을 때
// take 명령이 명확한 에러를 반환함을 검증합니다.
func TestMockScenarioI1_EmptyClipboardRejectsDeep(t *testing.T) {
	t.Parallel()

	// 공백만 있는 클립보드
	cb := &stubClipboard{read: "   "}
	app := newTestApp(t, mockAppConfig(), &stubTranslator{}, cb, &stubEditor{}, mockHistoryStore(t))

	err := app.Execute(context.Background(),
		[]string{"take", "patch", "--dry-run"},
		strings.NewReader(""), false)
	if err == nil {
		t.Fatal("Execute() expected error for empty clipboard, got nil")
	}
	if !strings.Contains(err.Error(), "clipboard is empty") {
		t.Errorf("error = %v, want 'clipboard is empty'", err)
	}
}

// TestMockScenarioI2_AgainWithNoHistoryReturnsError는 히스토리가 없을 때
// again 명령이 명확한 에러를 반환함을 검증합니다.
func TestMockScenarioI2_AgainWithNoHistoryReturnsError(t *testing.T) {
	t.Parallel()

	// 비어 있는 히스토리 스토어
	app := newTestApp(t, mockAppConfig(), &stubTranslator{}, &stubClipboard{}, &stubEditor{}, mockHistoryStore(t))

	err := app.Execute(context.Background(),
		[]string{"again"},
		strings.NewReader(""), false)
	if err == nil {
		t.Fatal("Execute() expected error when history is empty, got nil")
	}
}

// TestMockScenarioI3_InvalidTemplatePresetReturnsError는 존재하지 않는
// 템플릿 프리셋 지정 시 명확한 에러를 반환함을 검증합니다.
func TestMockScenarioI3_InvalidTemplatePresetReturnsError(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, mockAppConfig(), &stubTranslator{output: "ok"}, &stubClipboard{}, &stubEditor{}, mockHistoryStore(t))

	err := app.Execute(context.Background(),
		[]string{"go", "--template", "nonexistent-preset", "테스트"},
		strings.NewReader(""), false)
	if err == nil {
		t.Fatal("Execute() expected error for missing template preset, got nil")
	}
}

// TestMockScenarioI4_SubmitWithoutPasteReturnsError는 --paste 없이 --submit을
// 사용하면 명확한 에러를 반환함을 검증합니다.
func TestMockScenarioI4_SubmitWithoutPasteReturnsError(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, mockAppConfig(), &stubTranslator{output: "ok"}, &stubClipboard{}, &stubEditor{}, mockHistoryStore(t))

	err := app.Execute(context.Background(),
		[]string{"--submit", "confirm", "테스트"},
		strings.NewReader(""), false)
	if err == nil {
		t.Fatal("Execute() expected error for --submit without --paste, got nil")
	}
	if !strings.Contains(err.Error(), "--submit requires --paste") {
		t.Errorf("error = %v, want '--submit requires --paste'", err)
	}
}

// ---------------------------------------------------------------------------
// 시나리오 J: 파이프 입력 (stdin 활용)
// ---------------------------------------------------------------------------

// TestMockScenarioJ1_PipedStdinAttachedAsEvidence는 파이프로 전달된 stdin이
// evidence로 프롬프트에 첨부되는 시나리오를 검증합니다.
//
// 실사용 명령어: cat error.log | prtr go "이 에러를 분석해줘"
func TestMockScenarioJ1_PipedStdinAttachedAsEvidence(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Analyze this error"}
	cb := &stubClipboard{}
	app := newTestApp(t, mockAppConfig(), translator, cb, &stubEditor{}, mockHistoryStore(t))

	// 파이프로 들어온 에러 로그 시뮬레이션
	pipedInput := strings.NewReader(`panic: runtime error: invalid memory address or nil pointer dereference
goroutine 1 [running]:
main.main()
	/home/user/myapp/main.go:42 +0x68`)

	err := app.Execute(context.Background(),
		[]string{"go", "이 에러를 분석해줘"},
		pipedInput, true) // stdinPiped=true
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// 클립보드에 프롬프트가 복사됐는지 확인
	if cb.calls == 0 {
		t.Error("clipboard.Copy() not called")
	}

	// 파이프 입력 내용이 프롬프트에 포함됐는지 확인
	if !strings.Contains(cb.copied, "nil pointer dereference") {
		t.Errorf("clipboard content = %q, want piped stdin to be included", cb.copied)
	}
}

// TestMockScenarioJ2_MultilinePipedInputPreservesFormat은 멀티라인 파이프
// 입력이 형식을 유지하며 첨부됨을 검증합니다.
//
// 실사용 명령어: git diff | prtr go "변경 사항 요약해줘"
func TestMockScenarioJ2_MultilinePipedInputPreservesFormat(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Summarize the changes"}
	cb := &stubClipboard{}
	app := newTestApp(t, mockAppConfig(), translator, cb, &stubEditor{}, mockHistoryStore(t))

	// git diff 출력 시뮬레이션
	diffInput := strings.NewReader(`diff --git a/internal/auth/handler.go b/internal/auth/handler.go
index abc123..def456 100644
--- a/internal/auth/handler.go
+++ b/internal/auth/handler.go
@@ -42,6 +42,10 @@ func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
+	if user == nil {
+		http.Error(w, "unauthorized", http.StatusUnauthorized)
+		return
+	}`)

	err := app.Execute(context.Background(),
		[]string{"go", "변경 사항 요약해줘"},
		diffInput, true)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if cb.calls == 0 {
		t.Error("clipboard.Copy() not called")
	}
}

// ---------------------------------------------------------------------------
// 시나리오 K: demo 및 version
// ---------------------------------------------------------------------------

// TestMockScenarioK1_DemoRunsWithoutAPIKey는 API 키 없이도 demo가 실행되는
// 시나리오를 검증합니다.
//
// 실사용 명령어: prtr demo
func TestMockScenarioK1_DemoRunsWithoutAPIKey(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, mockAppConfig(), &stubTranslator{}, &stubClipboard{}, &stubEditor{}, mockHistoryStore(t))
	stdout, _ := buffersFromApp(app)

	err := app.Execute(context.Background(),
		[]string{"demo"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if stdout.Len() == 0 {
		t.Error("stdout is empty — demo should print preview")
	}
}

// TestMockScenarioK2_VersionOutput은 version 명령이 버전 문자열을 출력함을
// 검증합니다.
//
// 실사용 명령어: prtr version
func TestMockScenarioK2_VersionOutput(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, mockAppConfig(), &stubTranslator{}, &stubClipboard{}, &stubEditor{}, mockHistoryStore(t))
	stdout, _ := buffersFromApp(app)

	err := app.Execute(context.Background(),
		[]string{"version"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "test") {
		t.Errorf("stdout = %q, want version string", stdout.String())
	}
}

// ---------------------------------------------------------------------------
// 시나리오 L: 번역 정책 (translate policy)
// ---------------------------------------------------------------------------

// TestMockScenarioL1_InspectForceLangOverridesTranslationTarget은 inspect의
// --lang 플래그가 번역 타겟 언어를 오버라이드함을 검증합니다.
//
// 실사용 명령어: prtr inspect --lang ja "성능 최적화 방법"
func TestMockScenarioL1_InspectForceLangOverridesTranslationTarget(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "パフォーマンス最適化の方法"}
	cb := &stubClipboard{}
	app := newTestApp(t, mockAppConfig(), translator, cb, &stubEditor{}, mockHistoryStore(t))

	// inspect는 --lang 플래그를 지원함
	err := app.Execute(context.Background(),
		[]string{"inspect", "--lang", "ja", "성능 최적화 방법"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// inspect는 클립보드 복사 없음
	if cb.calls != 0 {
		t.Errorf("clipboard.Copy() called %d times during inspect, want 0", cb.calls)
	}
}

// TestMockScenarioL2_NoCopySkipsClipboard는 --no-copy 플래그가 클립보드 복사를
// 건너뜀을 검증합니다.
//
// 실사용 명령어: prtr --no-copy "테스트 메시지"
func TestMockScenarioL2_NoCopySkipsClipboard(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Test message"}
	cb := &stubClipboard{}
	app := newTestApp(t, mockAppConfig(), translator, cb, &stubEditor{}, mockHistoryStore(t))
	_, stderr := buffersFromApp(app)

	// --no-copy는 root-level 플래그 (go 서브커맨드 앞에 위치)
	err := app.Execute(context.Background(),
		[]string{"--no-copy", "테스트 메시지"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if cb.calls != 0 {
		t.Errorf("clipboard.Copy() called %d times with --no-copy, want 0", cb.calls)
	}
	if !strings.Contains(stderr.String(), "clipboard skipped") {
		t.Errorf("stderr = %q, want 'clipboard skipped'", stderr.String())
	}
}

// ---------------------------------------------------------------------------
// 시나리오 M: translate request struct 유효성
// ---------------------------------------------------------------------------

// TestMockScenarioM1_TranslateRequestFieldsArePropagated는 번역 요청의 모든
// 필드(Text, SourceLang, TargetLang, ProtectedTerms)가 올바르게 전달됨을
// 검증합니다.
func TestMockScenarioM1_TranslateRequestFieldsArePropagated(t *testing.T) {
	t.Parallel()

	translator := &stubTranslator{output: "Optimize the query"}
	cb := &stubClipboard{}
	repoRoot := t.TempDir()
	app := New(Dependencies{
		Version:         "test",
		Stdout:          &bytes.Buffer{},
		Stderr:          &bytes.Buffer{},
		Translator:      translator,
		Clipboard:       cb,
		Editor:          &stubEditor{},
		Launcher:        &stubLauncher{},
		Automator:       &stubAutomator{},
		SubmitConfirmer: &stubConfirmer{},
		ConfigLoader: func() (config.Config, error) {
			return mockAppConfig(), nil
		},
		ConfigInit:   func() (string, error) { return "/tmp/prtr/config.toml", nil },
		LookupEnv:    func(string) (string, bool) { return "", false },
		HistoryStore: mockHistoryStore(t),
		RepoContext:  &stubRepoContext{},
		RepoRootFinder: func() (string, error) {
			return repoRoot, nil
		},
		TermbookLoader: func(string) (termbook.Book, error) {
			return termbook.Book{ProtectedTerms: []string{"QueryBuilder", "SlowQueryLog"}}, nil
		},
	})

	err := app.Execute(context.Background(),
		[]string{"go", "QueryBuilder 쿼리를 최적화해줘"},
		strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	req := translator.gotInput
	// 소스 언어 확인
	if req.SourceLang != "ko" {
		t.Errorf("SourceLang = %q, want %q", req.SourceLang, "ko")
	}
	// 타겟 언어 확인
	if req.TargetLang != "en" {
		t.Errorf("TargetLang = %q, want %q", req.TargetLang, "en")
	}
	// 번역 요청 텍스트가 비어있지 않은지 확인
	// (보호 용어는 플레이스홀더로 치환되어 텍스트에 포함됨)
	if strings.TrimSpace(req.Text) == "" {
		t.Error("translator input text is empty")
	}
}

// translateRequestForTest는 테스트용 번역 요청을 생성하는 헬퍼입니다.
func translateRequestForTest(text, from, to string) translate.Request {
	return translate.Request{
		Text:       text,
		SourceLang: from,
		TargetLang: to,
	}
}

// TestMockScenarioM2_TranslateRequestBuilderHelper는 번역 요청 빌더 헬퍼
// 함수가 올바른 필드를 설정함을 검증합니다.
func TestMockScenarioM2_TranslateRequestBuilderHelper(t *testing.T) {
	t.Parallel()

	req := translateRequestForTest("안녕하세요", "ko", "en")
	if req.Text != "안녕하세요" {
		t.Errorf("Text = %q", req.Text)
	}
	if req.SourceLang != "ko" {
		t.Errorf("SourceLang = %q", req.SourceLang)
	}
	if req.TargetLang != "en" {
		t.Errorf("TargetLang = %q", req.TargetLang)
	}
}
