# prtr 실사용 가이드 — 명령어 전체 & Mock 테스트셋

> **대상**: prtr을 처음 사용하거나, 기능을 직접 검증해보고 싶은 개발자
> **버전**: v0.8.0
> **영문 참조**: [docs/guide.md](guide.md), [docs/reference.md](reference.md)

---

## 목차

1. [빠른 시작](#1-빠른-시작)
2. [전체 명령어 레퍼런스](#2-전체-명령어-레퍼런스)
3. [실사용 워크플로 시나리오](#3-실사용-워크플로-시나리오)
4. [Mock 테스트셋 실행 가이드](#4-mock-테스트셋-실행-가이드)
5. [Mock 입력 데이터](#5-mock-입력-데이터)
6. [설정 예시 (config.toml)](#6-설정-예시-configtoml)
7. [트러블슈팅](#7-트러블슈팅)

---

## 1. 빠른 시작

### 사전 요구사항

```bash
# Go 1.24.2 이상 필요
go version

# 소스에서 직접 빌드
git clone https://github.com/helloprtr/poly-prompt.git
cd poly-prompt
go build ./cmd/prtr
./prtr version
```

### API 키 없이 먼저 테스트

```bash
# API 키 없이 동작 확인
./prtr demo

# dry-run으로 프롬프트 미리보기
./prtr go "로그인 버그를 수정해줘" --dry-run

# 라우팅 경로만 확인 (클립보드 복사 없음)
./prtr inspect "코드 리뷰 요청"
```

### 초기 설정

```bash
# 안내형 첫 실행
./prtr start

# 상세 설정 (DeepL 키, 언어, 타겟 등)
./prtr setup

# 설정 확인
./prtr doctor
```

---

## 2. 전체 명령어 레퍼런스

### 핵심 전송 명령어

#### `prtr go` — AI에 요청 전송

```bash
# 기본 사용 (한국어 → 영어 자동 번역)
prtr go "로그인 버그를 수정해줘"

# 롤 지정 (be: 백엔드 엔지니어)
prtr go -r be "DB 연결 누수 분석해줘"

# 타겟 AI 지정
prtr go -t gemini "API 설계 검토"
prtr go -t codex "함수 구현"

# 템플릿 프리셋 지정
prtr go --template claude-structured "아키텍처 분석"
prtr go --template codex-implement "정렬 알고리즘 구현"

# 출력 언어 강제 지정
prtr go --lang ja "성능 최적화 방법"

# 클립보드 복사 생략 (터미널에만 출력)
prtr go --no-copy "간단한 질문"

# dry-run (전송 없이 미리보기)
prtr go --dry-run "테스트 메시지"

# 앱 실행 후 자동 붙여넣기 (Linux/macOS)
prtr go --launch "코드 분석"
prtr go --paste "긴 컨텍스트 분석"

# 파이프 입력 (stdin을 evidence로 첨부)
cat error.log | prtr go "이 에러를 분석해줘"
git diff | prtr go "변경 사항 요약"
```

#### `prtr again` — 최신 히스토리 재실행

```bash
# 마지막 요청 그대로 재전송
prtr again

# 다른 타겟으로 재전송
prtr again -t codex
```

#### `prtr swap` — 프로바이더 전환

```bash
# 최신 프롬프트를 gemini로 전환
prtr swap gemini

# claude로 전환
prtr swap claude

# codex로 전환
prtr swap codex
```

---

### 숏컷 명령어 (Shortcut)

프로젝트별 자주 쓰는 조합을 빠르게 실행:

```bash
# fix: codex + be 롤 + codex-implement 템플릿
prtr fix "nil pointer 에러 수정"

# ask: claude + claude-structured 템플릿
prtr ask "이 코드가 왜 느린가요?"

# review: claude + be 롤 + claude-review 템플릿
prtr review "PR #42 코드 리뷰"

# design: gemini + ui 롤 + gemini-stepwise 템플릿
prtr design "대시보드 UX 개선"
```

---

### take 명령어 (클립보드 → 다음 액션)

AI 응답을 클립보드에 복사한 후 실행:

```bash
# 패치 적용 프롬프트 생성
prtr take patch

# 테스트 계획 생성
prtr take test

# 커밋 메시지 생성
prtr take commit

# 디버그 분석 프롬프트
prtr take debug

# 리팩토링 계획
prtr take refactor

# 다른 타겟으로 재전송
prtr take patch -t gemini

# dry-run (클립보드 복사 없이 미리보기)
prtr take patch --dry-run
```

#### take --deep (5-worker 심층 분석)

```bash
# deep 분석 파이프라인 실행
prtr take patch --deep

# 특정 LLM 프로바이더 지정
prtr take patch --deep --llm claude
prtr take patch --deep --llm gemini
prtr take patch --deep --llm codex

# dry-run
prtr take patch --deep --dry-run

# 별칭: dip
prtr dip                    # = prtr take patch --deep
```

**deep 파이프라인 5단계:**
1. `planner` — 실행 계획 수립 (hard blocker)
2. `patcher` — 코드 변경 초안 작성 (hard blocker)
3. `critic` — 위험 요소 검토 (soft blocker)
4. `tester` — 테스트 계획 생성 (soft blocker)
5. `reconciler` — 최종 번들 패키징

**deep 실행 후 아티팩트:**
```
.prtr/runs/<run-id>/
├── manifest.json          # 실행 메타데이터
├── plan.json              # 실행 계획 및 Todo
├── events.jsonl           # 구조화된 이벤트 로그
├── source.md              # 클립보드 원문
├── evidence/
│   ├── repo_context.json  # git 컨텍스트
│   ├── history.json       # 최근 히스토리
│   └── git.diff          # 현재 변경사항
└── result/
    ├── patch_bundle.json  # 구조화된 패치 데이터
    ├── patch.diff         # 코드 변경사항 diff
    ├── tests.md           # 테스트 계획
    └── summary.md         # 실행 요약
```

---

### learn 명령어 (termbook)

```bash
# 프로젝트 용어 자동 학습 (히스토리 기반)
prtr learn

# dry-run (파일 작성 없이 미리보기)
prtr learn --dry-run

# 별칭
prtr marinate               # = prtr learn
```

학습된 용어는 `.prtr/termbook.toml`에 저장되어 이후 번역 시 보호됩니다.

---

### inspect 명령어 (미리보기)

```bash
# 라우팅 경로 미리보기 (클립보드 복사 없음)
prtr inspect "DB 인덱싱 최적화"

# 언어 강제 지정
prtr inspect --lang ja "쿼리 최적화"

# 별칭
prtr taste "코드 분석"      # = prtr inspect
```

---

### 캡슐 명령어 (work capsule)

```bash
# 현재 작업 상태 저장
prtr save
prtr save "auth-fix-v2"     # 레이블 지정
prtr save --note "로그인 버그 수정 중"

# 저장된 캡슐 목록
prtr list

# 최신 캡슐 상태 확인
prtr status

# 캡슐 복원 (ID 지정)
prtr resume <id>
prtr resume <id> --to gemini  # 다른 타겟으로 복원

# dry-run
prtr resume <id> --dry-run

# 오래된 캡슐 정리
prtr prune
prtr prune --older-than 30d
```

---

### 히스토리 명령어

```bash
# 최근 실행 목록
prtr history

# 검색
prtr history search "auth"

# 특정 항목 재실행
prtr rerun <id>

# 에디터에서 편집 후 재실행
prtr rerun <id> --edit

# 즐겨찾기 설정
prtr favorite <id>
prtr pin <id>
```

---

### 설정 명령어

```bash
# 초기 설정 (첫 실행)
prtr start
prtr prep                   # 별칭

# 상세 설정
prtr setup

# 언어 설정만 변경
prtr lang

# 진단 및 자동 수리
prtr doctor
prtr doctor --fix

# 버전 확인
prtr version

# 프로젝트 초기화 (.prtr/ 디렉토리 생성)
prtr init
```

---

### 템플릿 & 프로필 명령어

```bash
# 사용 가능한 템플릿 프리셋 목록
prtr templates list

# 특정 템플릿 내용 확인
prtr templates show claude-structured
prtr templates show codex-implement
prtr templates show gemini-stepwise

# 프로필 목록
prtr profiles list

# 프로필 상세 확인
prtr profiles show backend_review

# 프로필 적용 후 요청
prtr profiles use backend_review "코드 리뷰"
```

---

### 배경 watcher 명령어

```bash
# watcher 시작 (쉘 훅 자동 설치)
prtr watch start

# watcher 중지
prtr watch stop

# watcher 상태 확인
prtr watch status
```

---

### 특수 별칭 (v0.8.0)

| 별칭 | 원래 명령어 |
|------|------------|
| `dip` | `take patch --deep` |
| `taste` | `inspect` |
| `plate` | `swap` |
| `marinate` | `learn` |
| `prep` | `start` |

---

## 3. 실사용 워크플로 시나리오

### 시나리오 1: 버그 수정 (기본 루프)

```bash
# 1. 버그 설명을 Claude에 전송
prtr go "로그인 후 세션이 5분 만에 만료되는 버그 수정"

# 2. Claude 응답을 클립보드에 복사

# 3. 응답 기반으로 패치 프롬프트 생성
prtr take patch

# 4. Codex로 전환해서 구현 확인
prtr swap codex
```

### 시나리오 2: 심층 분석 (deep 파이프라인)

```bash
# 1. AI 응답을 클립보드에 복사 (코드 변경 제안 포함)
# 예: "internal/auth/handler.go의 42번 라인에서 nil 체크 누락..."

# 2. deep 분석 실행 (5단계 파이프라인)
prtr take patch --deep --dry-run

# 3. 아티팩트 확인
cat .prtr/runs/*/result/summary.md
cat .prtr/runs/*/result/patch.diff
```

### 시나리오 3: 다국어 개발팀 협업

```bash
# 한국어로 요청 → 영어로 번역 → Claude에 전달
prtr go "인증 미들웨어를 리팩토링해서 JWT 검증 로직을 분리해줘"

# 일본어 팀원을 위해 일본어로 전달
prtr go --lang ja "인증 미들웨어 리팩토링"

# 독일어로
prtr go --lang de "Auth-Middleware refactoring"
```

### 시나리오 4: 에러 로그 분석

```bash
# 방법 1: 파이프로 전달
cat /var/log/app.log | prtr go "이 에러 패턴 분석해줘"

# 방법 2: git diff와 함께
git diff HEAD~1 | prtr go "이 변경사항의 잠재적 버그 찾아줘"

# 방법 3: 파일 내용 직접 파이프
cat internal/auth/handler.go | prtr go "이 파일의 보안 취약점 분석"
```

### 시나리오 5: 작업 상태 저장 및 복원

```bash
# 작업 중 상태 저장
prtr save "auth-refactor-wip"

# 다른 브랜치에서 작업 후 복원
git checkout main
prtr list
prtr resume <id>

# drift 감지: 브랜치/커밋이 변경된 경우 경고 표시
```

---

## 4. Mock 테스트셋 실행 가이드

### 전체 테스트 실행

```bash
# 모든 테스트 실행
go test ./...

# 상세 출력
go test -v ./...

# 특정 패키지만
go test ./internal/app/...
go test ./internal/deep/...
go test ./internal/capsule/...
```

### 시나리오별 테스트 실행

```bash
# 시나리오 A: 기본 전송 루프
go test -v -run TestMockScenarioA ./internal/app/...

# 시나리오 B: 번역 파이프라인
go test -v -run TestMockScenarioB ./internal/app/...

# 시나리오 C: take 액션
go test -v -run TestMockScenarioC ./internal/app/...

# 시나리오 D: deep 파이프라인
go test -v -run TestMockScenarioD ./internal/app/...

# 시나리오 E: learn (termbook)
go test -v -run TestMockScenarioE ./internal/app/...

# 시나리오 F: inspect (미리보기)
go test -v -run TestMockScenarioF ./internal/app/...

# 시나리오 G: history & rerun
go test -v -run TestMockScenarioG ./internal/app/...

# 시나리오 H: shortcut 명령어
go test -v -run TestMockScenarioH ./internal/app/...

# 시나리오 I: 에러 처리
go test -v -run TestMockScenarioI ./internal/app/...

# 시나리오 J: 파이프 입력 (stdin)
go test -v -run TestMockScenarioJ ./internal/app/...

# 시나리오 K: demo & version
go test -v -run TestMockScenarioK ./internal/app/...

# 시나리오 L: 번역 정책
go test -v -run TestMockScenarioL ./internal/app/...

# 시나리오 M: translate request 유효성
go test -v -run TestMockScenarioM ./internal/app/...
```

### deep 파이프라인 통합 테스트

```bash
# 시나리오 1: 모든 단계 완료 (Happy Path)
go test -v -run TestScenario1 ./internal/deep/...

# 시나리오 2: 장애 복원력 (Resilience)
go test -v -run TestScenario2 ./internal/deep/...

# 시나리오 3: 스키마 및 아티팩트 무결성
go test -v -run TestScenario3 ./internal/deep/...

# worker 그래프 테스트
go test -v ./internal/deep/worker/...
```

### 캡슐 시스템 테스트

```bash
go test -v ./internal/capsule/...
```

### 특정 테스트 하나만 실행

```bash
# 한국어 프롬프트 전송 테스트
go test -v -run TestMockScenarioA1_GoSendsKoreanPrompt ./internal/app/

# deep 파이프라인 전체 실행 테스트
go test -v -run TestMockScenarioD1_TakeDeepPatchWithRepoContext ./internal/app/

# 빈 클립보드 에러 테스트
go test -v -run TestMockScenarioI1_EmptyClipboardRejectsDeep ./internal/app/
```

### 병렬 실행 (빠른 CI)

```bash
# 모든 테스트 병렬 실행
go test -parallel 8 ./...

# 특정 패키지 병렬
go test -parallel 4 ./internal/app/... ./internal/deep/...
```

### 테스트 커버리지 확인

```bash
# 커버리지 리포트 생성
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# 터미널에서 커버리지 요약
go tool cover -func=coverage.out | grep -E "total|mock"
```

---

## 5. Mock 입력 데이터

### Mock 한국어 프롬프트 예시

테스트나 demo에 사용할 수 있는 실제 프롬프트 예시입니다:

#### 버그 수정 요청

```
nil pointer dereference 에러가 internal/auth/handler.go 87번 라인에서 발생합니다.
토큰이 만료됐을 때 user 객체가 nil인지 확인하지 않아서 발생하는 문제입니다.
```

#### 기능 구현 요청

```
JWT refresh token 교체 메커니즘을 구현해줘.
- access token: 15분 유효
- refresh token: 7일 유효
- refresh 시 이전 refresh token은 즉시 무효화 (rotation)
- Redis에 토큰 블랙리스트 저장
```

#### 코드 리뷰 요청

```
internal/database/pool.go의 커넥션 풀 구현을 리뷰해줘.
특히 다음 사항을 중점적으로 봐줘:
- 고루틴 누수 가능성
- 컨텍스트 취소 처리
- 에러 핸들링 일관성
```

#### 리팩토링 요청

```
UserService를 리팩토링해서 의존성 주입 패턴을 적용해줘.
현재는 전역 DB 인스턴스를 직접 참조하고 있어서 테스트가 어려운 상태야.
```

### Mock 클립보드 내용 (take 명령어용)

AI가 응답한 내용을 클립보드에 복사한 상황을 시뮬레이션:

#### patch 액션용

```
The bug is in internal/auth/handler.go at line 87.
The fix requires adding a nil check before accessing the user object:

if user == nil {
    return nil, ErrUnauthorized
}

Additionally, the token validation should happen before the user lookup,
not after, to prevent unnecessary database queries.
```

#### test 액션용

```
The rate limiting middleware has been implemented using a token bucket algorithm
with Redis as the backend store. Key implementation details:
- Bucket capacity: 100 requests
- Refill rate: 10 requests/second
- Redis key format: rate_limit:{user_id}:{endpoint}
- TTL: automatically set to match the window duration
```

#### commit 액션용

```
Added exponential backoff retry logic to the HTTP client:
- Maximum 3 retry attempts
- Initial delay: 100ms, doubled each attempt
- Added jitter (±10%) to prevent thundering herd
- Respects context cancellation during retry wait
```

### Mock 에러 로그 (파이프 입력용)

```bash
# 사용 방법: echo 또는 파일로 파이프
echo "panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x1 addr=0x0 pc=0x68b4a3]

goroutine 1 [running]:
github.com/myapp/internal/auth.(*Handler).Login(0xc000124000, 0x15fce00, 0xc000126000, 0xc0001a6000)
	/home/user/myapp/internal/auth/handler.go:87 +0x63
net/http.HandlerFunc.ServeHTTP(0x12e5b40, 0x15fce00, 0xc000126000, 0xc0001a6000)
	/usr/local/go/src/net/http/server.go:2047 +0x44" | prtr go "이 패닉 에러를 분석해줘"
```

---

## 6. 설정 예시 (config.toml)

`~/.config/prtr/config.toml` 설정 파일 예시:

### 한국어 개발자 기본 설정

```toml
deepl_api_key = "YOUR-DEEPL-API-KEY"
translation_source_lang = "ko"
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
paste_delay_ms = 700
submit_mode = "manual"

[launchers.gemini]
command = "gemini"
paste_delay_ms = 700
submit_mode = "manual"

[launchers.codex]
command = "codex"
paste_delay_ms = 800
submit_mode = "manual"
```

### 롤(Role) 설정 예시

```toml
[roles.be]
prompt = "Expert Backend Engineer & Tech Lead"

  [roles.be.targets.claude]
  prompt = "You are an expert Backend Engineer and Tech Lead specializing in Go microservices."
  template_preset = "claude-structured"

  [roles.be.targets.codex]
  prompt = "// Role: Expert Backend Engineer"
  template_preset = "codex-implement"

[roles.fe]
prompt = "Expert Frontend Engineer"

  [roles.fe.targets.claude]
  prompt = "You are an expert Frontend Engineer specializing in React and TypeScript."
  template_preset = "claude-structured"

[roles.review]
prompt = "Expert Code Reviewer"

  [roles.review.targets.claude]
  prompt = "You are an expert code reviewer. Focus on: correctness, security, performance, maintainability."
  template_preset = "claude-review"
```

### 숏컷(Shortcut) 설정 예시

```toml
[shortcuts.fix]
target = "codex"
role = "be"
template_preset = "codex-implement"

[shortcuts.ask]
target = "claude"
template_preset = "claude-structured"

[shortcuts.review]
target = "claude"
role = "review"
template_preset = "claude-review"

[shortcuts.design]
target = "gemini"
role = "ui"
template_preset = "gemini-stepwise"

[shortcuts.debug]
target = "claude"
role = "be"
template_preset = "claude-structured"
```

---

## 7. 트러블슈팅

### 클립보드 문제

```bash
# 진단
prtr doctor

# Linux에서 클립보드 도구 설치
sudo apt install xclip        # X11
sudo apt install wl-clipboard  # Wayland
```

### 번역이 안 될 때

```bash
# DeepL API 키 확인
prtr doctor

# 키 직접 설정
prtr setup

# API 키 없이 사용 (영어 입력 시 번역 생략)
prtr go "Fix the login bug"  # 영어면 번역 생략됨
```

### deep 실행 실패

```bash
# 아티팩트 확인
ls .prtr/runs/

# 이벤트 로그 확인
cat .prtr/runs/*/events.jsonl | jq .

# 특정 worker 결과 확인
cat .prtr/runs/*/workers/planner/result.json
cat .prtr/runs/*/workers/patcher/result.json
```

### 설정 초기화

```bash
# 진단 후 자동 수리 시도
prtr doctor --fix

# 설정 파일 위치 확인
ls ~/.config/prtr/

# 설정 재생성
prtr setup
```

### 테스트 실패 시

```bash
# 포맷 검사
gofmt -l $(git ls-files '*.go')

# 포맷 자동 수정
gofmt -w $(git ls-files '*.go')

# 전체 테스트
go test ./...

# 상세 실패 원인 확인
go test -v -run TestMockScenario ./internal/app/ 2>&1 | head -50
```

---

## 참고 링크

- **영문 가이드**: [docs/guide.md](guide.md)
- **한국어 가이드**: [docs/guide.ko.md](guide.ko.md)
- **명령어 레퍼런스**: [docs/reference.md](reference.md)
- **변경 이력**: [CHANGELOG.md](../CHANGELOG.md)
- **설치 가이드**: [INSTALLATION.md](../INSTALLATION.md)
- **이슈 제보**: https://github.com/helloprtr/poly-prompt/issues
