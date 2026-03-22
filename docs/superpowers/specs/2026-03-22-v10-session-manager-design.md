# Design Spec: prtr v1.0 — AI Work Session Manager

**Date:** 2026-03-22
**Status:** Draft
**Scope:** Full architectural redesign — Session as first-class citizen, subprocess model, multi-model handoff, simplified command surface

---

## Problem

prtr 0.x는 **프롬프트 조립 도구**다. 사용자가 메시지를 작성하면 prtr이 구조화해서 AI 앱에 붙여넣는다. 이 모델은 두 가지 근본적인 한계를 가진다.

1. **반복 비용.** 새 AI 세션을 열 때마다 사용자는 같은 맥락을 다시 설명해야 한다. 어떤 파일인지, 무엇을 하려는지, 어떤 제약이 있는지 — 매번 처음부터.

2. **모델 전환 시 기억 손실.** Claude Code에서 토큰 한계에 도달해 Gemini로 옮기면, 이전 세션에서 무엇을 했는지 사라진다. "어디까지였지?" 를 다시 설명해야 한다.

두 문제의 공통 원인: prtr이 **작업 세션**을 모르고, **프롬프트 하나**만 안다.

---

## Goals

- 사용자가 AI 모델을 바꿔도 **작업 의도와 코드 변화**가 유지된다
- 세션을 닫고 다음날 `prtr`을 열면 이전 작업을 이어서 할 수 있다
- 같은 작업을 다른 모델에 핸드오프할 수 있다
- 명령어가 단순해서 처음 쓰는 사람도 바로 쓸 수 있다

## Non-Goals

- AI API 직접 호출 (요약, 추론 등 — prtr은 AI가 아님)
- 전체 대화 기록 캡처 (PTY wrapping, 전체 대화 재생)
- 병렬 다중 모델 동시 실행 (v1.0 범위 밖, 별도 스펙)
- GUI AI 앱 자동화 (Claude.app, 브라우저) — 이 스펙은 TUI AI에 집중
- 기존 0.x 명령어 즉시 제거 (하위 호환 유지)

---

## Core Concept: Session as First-Class Citizen

0.x: **프롬프트**가 1등 시민. 세션은 개념 없음.
1.0: **세션**이 1등 시민. 프롬프트는 세션에서 파생된다.

세션은 "이 작업에 대해 prtr이 기억하는 것" 전체다. 사용자는 세션을 명시적으로 관리하지 않는다 — prtr이 알아서 만들고, 업데이트하고, 보존한다.

**핸드오프 품질에 대한 솔직한 설명:**
prtr이 자동으로 유지하는 것은 "작업 의도 + 코드 변화"다. 전체 대화 기록은 아니다. 핸드오프 프롬프트는 원래 목표, git diff, 마지막 AI 응답으로 구성된다. 중간 결정 과정이나 시도한 접근법은 `prtr checkpoint`로 메모를 남겨야 보존된다. checkpoint 없이도 기본 핸드오프는 동작하지만, 복잡한 세션일수록 checkpoint가 품질을 높인다.

---

## 1. Session State

**위치:** `~/.config/prtr/sessions/<repo-hash>-<session-id>.json`

repo-hash: `git rev-parse --show-toplevel`의 절대 경로를 SHA256으로 해시한 8자리

```json
{
  "id": "s_abc123",
  "repo": "/Users/koo/dev/myapp",
  "repo_hash": "d4e5f6a7",
  "task_goal": "인증 미들웨어 리팩토링",
  "files": ["auth/*.ts"],
  "mode": "edit",   // "review" | "edit" | "fix" | "design"
  "constraints": ["TypeScript", "테스트 포함"],
  "target_model": "claude",
  "status": "active",
  "started_at": "2026-03-22T10:00:00Z",
  "last_activity": "2026-03-22T11:30:00Z",
  "base_git_sha": "abc123def456",
  "checkpoints": [
    {
      "note": "JWT 검증 완료, refresh token 미완",
      "git_sha": "def456abc789",
      "at": "2026-03-22T11:00:00Z"
    }
  ]
}
```

**세션이 담는 것:**
- 사용자 의도: 목표, 파일, 모드, 제약 (세션 시작 시 구조화)
- `base_git_sha`: `git rev-parse HEAD`를 세션 **생성 시점**에 캡처. 세션 시작 전부터 있던 uncommitted changes는 diff에 포함되며 이는 의도된 동작
- 체크포인트 메모: 사용자가 선택적으로 남기는 진행 상황 메모

**세션이 담지 않는 것:**
- 전체 대화 기록
- AI 응답 내용 (글로벌 `last-response.json`에서 읽음 — v0.9 아키텍처 유지)
- AI API 호출 결과

**세션 status 값:** `active` | `completed`

`active`: 현재 진행 중. `completed`: `prtr done`으로 종료됨. completed 세션은 삭제되지 않으며 `prtr sessions`에서 조회 가능. v1.0에서 별도 archived 상태는 없음 — completed가 최종 상태다.

---

## 2. Session Lifecycle

### 2.1 현재 세션 결정 방식

prtr은 명령 실행 시 다음 순서로 "현재 세션"을 결정한다:

1. `git rev-parse --show-toplevel`로 현재 repo 경로를 구하고 repo-hash를 계산
2. `~/.config/prtr/sessions/` 에서 해당 repo-hash의 `status: active` 세션을 찾음
3. active 세션이 정확히 하나 → 그 세션이 현재 세션
4. active 세션이 여러 개 → `last_activity` 기준 최신 세션을 현재 세션으로 선택
5. active 세션이 없음 → 새 세션 생성 플로우 실행
6. git repo가 아닌 디렉터리 → 세션 기능 비활성화, 프롬프트 조립만 동작 (0.x 모드로 폴백)

### 2.2 `prtr` (bare) 명령 결정 트리

`prtr`을 아무 인자 없이 실행했을 때의 두 가지 경로:

**경로 A — active 세션이 있을 때 (섹션 2.1 결과):**
```
$ prtr
─────────────────────────────────────
이어서 할까요?

"인증 미들웨어 리팩토링" — 어제 오후 3시 (Claude)
변경된 파일: auth/jwt.ts, auth/middleware.ts
─────────────────────────────────────
[Enter] 이어서  [n] 새 작업
```
Enter → 핸드오프 프롬프트 구성 후 AI 재실행 (섹션 2.6 참고)
n → 경로 B 실행

**경로 B — active 세션이 없을 때 (또는 사용자가 "새 작업" 선택):**
```
$ prtr
무엇을 하려 하나요? 인증 미들웨어 리팩토링
어떤 파일? (Enter로 건너뜀) auth/*.ts
모드: [1] review  [2] edit  [3] fix  [4] design (기본: edit) 2
제약조건? (Enter로 건너뜀) TypeScript, 테스트 포함
```
입력 완료 → 세션 생성 → AI 실행

**필수/선택 필드:**
- `task_goal`: 필수 (빈 입력 불가)
- `files`: 선택 (Enter로 건너뛰면 빈 배열)
- `mode`: 선택 (기본값: `edit`)
- `constraints`: 선택 (Enter로 건너뛰면 빈 배열)

### 2.3 모드별 명령으로 세션 생성

```bash
prtr review auth/*.ts   # → mode: review, files: [auth/*.ts], task_goal 질문
prtr fix "로그인 버그"  # → mode: fix, task_goal: "로그인 버그", files 질문
prtr edit               # → mode: edit, task_goal과 files 질문
```

모드별 명령은 mode와 첫 번째 인자를 미리 채운 채로 나머지 필드만 질문한다.

**이미 active 세션이 있을 때 (모드별 명령 포함):**
```
⚡ 진행 중인 세션: "인증 미들웨어 리팩토링" (2시간 전)
이어서 할까요, 새로 시작할까요? [이어서/새로]
```

### 2.4 AI TUI 실행 (subprocess model)

prtr이 AI CLI 도구를 subprocess로 실행한다. 프롬프트 전달은 **클립보드 + 실행** 방식으로 기존 v0.8/v0.9 아키텍처를 유지한다.

**프롬프트 전달 방식:**
1. 구조화된 시작/핸드오프 프롬프트를 클립보드에 쓴다
2. AI TUI 바이너리를 `os.StartProcess`로 실행한다 (PTY 할당 포함)
3. prtr은 프로세스가 종료될 때까지 대기한다 (Wait)
4. 사용자가 TUI 안에서 클립보드 붙여넣기로 첫 프롬프트를 전달한다

이 방식은 기존 v0.9 clipboard watcher 및 watch 훅과 호환된다. subprocess로 실행하면 프로세스 종료 이벤트를 prtr이 직접 감지할 수 있어 종료 시 자동 캡처가 가능하다.

**AI 바이너리 이름 매핑:**

| `@model` | 바이너리 후보 (순서대로 탐색) |
|----------|------------------------------|
| `@claude` (기본) | `claude` |
| `@gemini` | `gemini`, `gemini-cli` |
| `@codex` | `codex` |

prtr은 시작 시 `$PATH`에서 각 바이너리를 탐색한다. 바이너리를 찾지 못하면: `gemini-cli not found. Install it first: ...` 출력 후 종료.

**git이 없는 디렉터리:** 세션 생성 없이 프롬프트 조립 + 클립보드 전달만 수행 (0.x 폴백 모드).

### 2.5 종료 시 자동 캡처

TUI 프로세스가 종료되면 prtr이 자동으로:

1. `git diff <base_git_sha>` 캡처 → 이 세션에서 바뀐 코드
2. `~/.config/prtr/last-response.json` 읽기 → 마지막 AI 응답 (v0.8/v0.9 watch 훅 결과)
3. 세션 파일 업데이트 (`last_activity` 갱신)
4. 터미널에 한 줄:
   ```
   ✓ 세션 저장됨 — 다음에 prtr로 이어서
   ```

AI 호출 없음. git diff + 마지막 응답 + 체크포인트 메모가 세션 상태 전체다.

### 2.6 복원 — 핸드오프 프롬프트 구성

섹션 2.2 경로 A에서 "이어서"를 선택하면 prtr이 핸드오프 프롬프트를 구성해서 AI를 재실행한다:

```
[작업 목표]
인증 미들웨어 리팩토링 (TypeScript, 테스트 포함)

[파일 범위]
auth/*.ts

[이전 세션 진행 상황]
체크포인트: "JWT 검증 완료, refresh token 미완"
코드 변화 (auth/jwt.ts +47 -12, auth/middleware.ts +23 -5):
<diff 요약>

[마지막 AI 응답]
...JWT 검증 로직은 완료됐고, refresh token 처리를 다음에...

이어서 작업해주세요.
```

### 2.7 종료

```bash
prtr done               # 세션 완료 처리 → status: completed
```

completed 세션은 삭제되지 않는다. `prtr sessions`로 조회 가능. v1.0에서 completed가 최종 상태다.

---

## 3. Handoff

### 3.1 다른 모델로 전환

```bash
prtr @gemini
```

현재 세션 상태를 Gemini CLI에 전달하고 실행한다. Claude에서 작업하다가 토큰 한계에 도달하거나 다른 모델로 접근을 바꾸고 싶을 때.

**active 세션이 없을 때:** `No active session. Run prtr review|edit|fix|design first.` 출력 후 종료.

**핸드오프 프롬프트 구성 (API 호출 없음):**

| 요소 | 출처 | 역할 |
|------|------|------|
| 목표 + 파일 + 모드 + 제약 | 세션 start 시 입력 | "무엇을 하려 했나" |
| `git diff <base_sha>` | 자동 캡처 | "실제로 무엇이 바뀌었나" |
| 체크포인트 메모들 | 사용자 선택적 입력 | "중간 결정 사항" |
| `last-response.json` | watch 훅 캡처 | "AI가 마지막으로 뭐라 했나" |

### 3.2 병렬 다중 모델 비교 (Future Work)

같은 구조화된 작업을 여러 모델에 동시에 보내서 결과를 비교하는 기능. v1.0 범위 밖. 별도 스펙으로 다룬다.

---

## 4. Command Surface

### 4.1 주요 명령 (사용자가 알아야 할 것)

```bash
prtr                     # 현재 세션 이어서. 없으면 안내 후 시작.
prtr review [files]      # 코드 리뷰 세션 시작
prtr edit [files]        # 코드 수정 세션 시작 (기본 모드)
prtr fix [description]   # 버그 수정 세션 시작
prtr design [topic]      # 설계/아키텍처 세션 시작

prtr @gemini             # 현재 세션을 Gemini로 핸드오프
prtr @codex              # 현재 세션을 Codex로 핸드오프
```

파일이나 설명 없이 모드 명령만 실행하면 prtr이 한 번씩 질문한다:
```
$ prtr review
어떤 파일을 리뷰할까요? auth/*.ts
```

### 4.2 보조 명령 (가끔 씀)

```bash
prtr status              # 현재 세션 상태 + Work Capsule drift 정보
prtr sessions            # 과거 세션 목록 (completed 포함)
prtr checkpoint "메모"   # 진행 상황 메모 (선택 — 핸드오프 품질 향상)
prtr done                # 세션 완료 처리
prtr inspect             # 생성될 프롬프트 미리보기 (dry-run)
```

### 4.3 유틸

```bash
prtr setup               # 초기 설정
prtr doctor              # 환경 진단 (AI 바이너리 탐색 포함)
prtr version             # 버전 확인
```

### 4.4 레거시 (숨김, 하위 호환)

help에 표시되지 않지만 동작은 유지된다. **레거시 명령은 세션을 생성하지 않는다** — 기존 0.x 동작을 그대로 수행한다. 기존 사용자 스크립트/습관 보호.

```bash
go, swap, take, again, start, learn
resume   # v0.9 "대화 이어서" 동작 유지 (세션 핸드오프와 별개)
```

---

## 5. Session Modes

세션 모드는 시작 프롬프트 구조에 영향을 준다.

| 모드 | 명령 | 프롬프트 포커스 |
|------|------|----------------|
| `review` | `prtr review` | 코드 품질, 잠재 이슈, 개선 제안 |
| `edit` | `prtr edit` (또는 기본값) | 구현, 코드 수정 |
| `fix` | `prtr fix` | 버그 원인 파악 + 수정 |
| `design` | `prtr design` | 아키텍처, 설계 결정 |

파일 지정 없이 `prtr`만 실행 시 기본 모드는 `edit`이다.

---

## 6. prtr status

`prtr status`는 두 정보를 함께 보여준다:

```
$ prtr status

[현재 세션]
작업: 인증 미들웨어 리팩토링
파일: auth/*.ts
모드: edit
시작: 2시간 전 (Claude)
변경: auth/jwt.ts +47, auth/middleware.ts +23
체크포인트: "JWT 검증 완료, refresh token 미완"

[Work Capsule]
마지막 캡슐: v3 "jwt-done" (1시간 전)
Drift: auth/refresh.ts — 저장 이후 변경됨
```

Work Capsule 정보는 v0.8에서 `prtr status`가 보여주던 내용을 그대로 유지한다.

---

## 7. New Files and Paths

```
~/.config/prtr/sessions/                       # 세션 파일 디렉터리 (신규)
~/.config/prtr/sessions/<hash>-<id>.json       # 세션 파일 (신규)
~/.config/prtr/last-response.json              # 기존 유지 (v0.8/v0.9)
~/.config/prtr/clipboard-watcher.pid           # 기존 유지 (v0.9)
```

---

## 8. Work Capsule과의 관계

v0.8 Work Capsule (`prtr save/list/prune`)은 "코드 상태 스냅샷"이다. v1.0 Session은 "작업 의도 + 진행 상황"이다. 두 개념이 보완적이다.

**v1.0에서 각 명령의 위치:**

| 명령 | v1.0 상태 |
|------|-----------|
| `prtr save` | 유지 (Work Capsule 스냅샷) |
| `prtr list` | 유지 (Capsule 목록) |
| `prtr prune` | 유지 (Capsule 정리) |
| `prtr status` | Session + Capsule 통합 표시 |
| `prtr resume --to` | `prtr @model`로 대체 (레거시 alias 유지) |

Work Capsule과 Session의 완전 통합은 별도 스펙으로 다룬다.

---

## 9. Backward Compatibility

- 0.x 레거시 명령어 (`go`, `swap`, `take`, `again`, `start`, `learn`, `resume`)는 숨겨진 alias로 유지, **세션 생성 없이** 0.x 동작 수행
- v0.9 `prtr resume [message]`는 레거시 alias로 유지 — v1.0에서는 `prtr`로 이어서 하는 것이 동등한 기능
- 설정 파일 포맷 변경 없음
- `last-response.json`, watch 훅 기존 경로 유지
- Homebrew tap 배포 방식 유지, 버전만 1.0.0으로 변경

---

## 10. Open Questions

1. **parallel @model 실행 UX:** 같은 작업을 두 모델에 동시에 보낼 때 두 TUI를 어떻게 나란히 보여줄지. tmux split? 새 탭? 별도 스펙 필요.

2. **Work Capsule 완전 통합 시점:** Session과 Capsule을 하나의 개념으로 합칠지, 별도로 유지할지.

3. **GUI AI (Claude.app) 지원:** 이 스펙은 TUI만 다룸. GUI 지원은 별도 스펙.

4. **non-git 디렉터리 동작 범위:** git 없으면 0.x 폴백 모드. 사용자에게 명시적으로 알릴지, 조용히 폴백할지.

5. **바이너리 이름 충돌 처리:** `gemini`, `gemini-cli` 둘 다 있을 때 우선순위. 사용자가 `prtr setup`에서 직접 지정할 수 있게 할지.
