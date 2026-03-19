# prtr v0.8.1 상세 사용 설명서 (한국어)

이 문서는 `v0.8.1` 태그 기준으로 `prtr`를 처음부터 끝까지 사용할 수 있게 정리한 상세 가이드입니다.

`prtr`는 터미널과 AI 도구(Claude / Codex / Gemini) 사이에 위치하는 **AI 작업 커맨드 레이어**입니다.  
로그, diff, 의도를 입력하면 `prtr`가 프롬프트를 조합하고, 앱별 형식에 맞게 다듬고, 필요하면 앱 실행과 붙여넣기까지 이어 줍니다.

v0.8.1에서 특히 중요한 것은 두 가지입니다.

- `take --deep`: 다단계 딥 파이프라인
- `save / resume`: 세션을 닫아도 이어서 작업할 수 있는 Work Capsule

---

## 목차

1. `prtr`가 하는 일
2. 설치와 첫 실행
3. 가장 먼저 익혀야 할 핵심 루프
4. 첫 요청 보내기: `prtr go`
5. 같은 요청 반복/비교: `again`, `swap`
6. 답변을 다음 액션으로 바꾸기: `take`
7. 더 깊게 처리하기: `take --deep`, `--llm`
8. 프로젝트 용어 보호: `learn`
9. 보내기 전에 확인하기: `inspect`
10. 작업 상태 저장과 복원: `save`, `resume`, `status`, `list`, `prune`
11. 셸 감시기: `watch`
12. 설정 파일과 환경 변수
13. 히스토리와 재실행
14. 숨겨진 별칭(이스터에그)
15. 추천 워크플로
16. 문제 해결

---

## 1. `prtr`가 하는 일

`prtr`는 "첫 프롬프트 생성기"보다 더 넓은 도구입니다.

- 내가 쓴 한국어 의도나 로그를 받아 프롬프트를 만듭니다.
- Claude, Codex, Gemini 중 원하는 대상에 맞게 프롬프트를 정리합니다.
- 바로 다른 AI 앱으로 비교할 수 있게 도와줍니다.
- AI가 준 답변을 다시 다음 액션 프롬프트로 바꿉니다.
- 프로젝트 고유 용어를 보호합니다.
- 딥 파이프라인으로 plan / diff / test plan / summary artifact를 남깁니다.
- v0.8부터는 작업 상태를 저장해 나중에 이어서 쓸 수 있습니다.

핵심 루프는 이렇게 이해하면 됩니다.

```text
go -> swap / again -> take -> save / resume
```

실무에서 자주 쓰는 루프는 보통 이렇습니다.

```bash
npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘"
prtr swap gemini
prtr take patch --deep --llm=claude
prtr save "test fix midpoint"
```

---

## 2. 설치와 첫 실행

### macOS

```bash
brew tap helloprtr/homebrew-tap
brew install prtr
```

### Linux / Windows

GitHub Releases에서 바이너리를 받아 `PATH`에 올리면 됩니다.

### 설치 확인

```bash
prtr version
prtr demo
prtr go "explain this error" --dry-run
```

처음에는 API 키 없이 아래 두 경로를 먼저 확인하는 것이 좋습니다.

- `prtr demo`
- 영어 입력으로 `--dry-run`

### 첫 실사용

```bash
prtr start
```

`start`는 초보자용 진입점입니다.

- 설정이 비어 있으면 온보딩 질문을 진행합니다.
- `doctor`를 먼저 돌려 환경을 점검합니다.
- 첫 요청까지 이어 줍니다.

보다 자세한 기본값 설정은 아래를 사용합니다.

```bash
prtr setup
```

언어 설정만 바꾸고 싶다면:

```bash
prtr lang
```

---

## 3. 가장 먼저 익혀야 할 핵심 루프

### 1단계: 첫 질문

```bash
prtr go fix "왜 테스트가 깨지는지 정확한 원인만 찾아줘"
```

### 2단계: 다른 AI와 비교

```bash
prtr swap codex
prtr swap gemini
```

### 3단계: 답변을 다음 액션으로 변환

```bash
prtr take patch
prtr take test
```

### 4단계: 깊은 실행 파이프라인 사용

```bash
prtr take patch --deep --llm=claude
```

### 5단계: 작업 상태 저장

```bash
prtr save "login bug midpoint"
```

### 6단계: 나중에 이어서 재개

```bash
prtr resume
prtr resume --to codex
```

---

## 4. 첫 요청 보내기: `prtr go`

`go`는 가장 중요한 기본 명령입니다.

```bash
prtr go "이 에러 원인 분석해줘"
```

### 모드 사용

```bash
prtr go fix "왜 테스트가 깨지는지 정확한 원인만 찾아줘"
prtr go review "이 PR에서 위험한 부분만 짚어줘"
prtr go design "이 기능 구조 설계해줘"
```

모드 의미:

- `fix`: 버그 원인 파악
- `review`: 위험도, 회귀, 누락 확인
- `design`: 구조와 설계 관점

### 대상 앱 지정

```bash
prtr go "이 구조 문제점 봐줘" --to claude
prtr go "이 함수 개선해줘" --to codex
prtr go review "이 설계 위험도 평가해줘" --to gemini
```

지정하지 않으면 기본 target 또는 마지막 target을 사용합니다.

### 전송 전 편집

```bash
prtr go design "이 기능 구조 설계해줘" --edit
```

### 미리 보기만

```bash
prtr go "이 문장 설명해줘" --dry-run
```

### 파이프 입력

```bash
npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘"
cat crash.log | prtr go fix
cat internal/api.go | prtr go review "이 파일의 위험한 부분을 찾아줘"
```

입력 규칙:

- 메시지만 주면 메시지가 프롬프트 본문입니다.
- stdin만 주면 stdin이 프롬프트 본문입니다.
- 메시지와 stdin을 함께 주면 메시지가 본문, stdin은 evidence가 됩니다.

### 자동 컨텍스트

Git 저장소 안에서 실행하면 자동으로 다음이 붙습니다.

- repo 이름
- 현재 branch
- 변경 파일 요약
- `learn`으로 저장한 보호 용어

끄고 싶다면:

```bash
prtr go "이 요청만 그대로 써줘" --no-context
```

---

## 5. 같은 요청 반복/비교: `again`, `swap`

### `prtr again`

직전 흐름을 같은 설정으로 다시 실행합니다.

```bash
prtr again
prtr again --edit
prtr again --dry-run
```

이럴 때 좋습니다.

- 방금 보낸 요청을 조금만 다르게 다시 보내고 싶을 때
- 최종 프롬프트를 편집해 재전송하고 싶을 때

### `prtr swap`

직전 요청을 다른 AI 앱용으로 다시 조합합니다.

```bash
prtr swap claude
prtr swap codex
prtr swap gemini
prtr swap gemini --edit
prtr swap claude --dry-run
```

이럴 때 좋습니다.

- Claude와 Gemini를 같은 요청으로 비교하고 싶을 때
- Codex 스타일 구현 응답이 더 잘 맞는지 확인하고 싶을 때

---

## 6. 답변을 다음 액션으로 바꾸기: `take`

`take`는 **클립보드에 복사한 AI 답변**을 읽어 다음 액션용 새 프롬프트로 바꿉니다.

```bash
prtr take patch
prtr take test --to codex
prtr take commit --dry-run
prtr take summary --edit
```

지원 액션:

- `patch`
- `test`
- `commit`
- `summary`
- `clarify`
- `issue`
- `plan`
- `debug`
- `refactor`

각 액션 의미:

- `patch`: 답변을 구현 요청으로 바꿈
- `test`: 테스트 작성 요청으로 바꿈
- `commit`: 커밋 메시지 요청으로 바꿈
- `summary`: 재사용 가능한 요약으로 바꿈
- `clarify`: 더 쉬운 설명을 요구하는 후속 질문으로 바꿈
- `issue`: 이슈/태스크 형식으로 바꿈
- `plan`: 단계별 계획으로 바꿈
- `debug`: 원인 파악 + 재현 + 수정 방향 요청으로 바꿈
- `refactor`: 안전한 리팩터링 요청으로 바꿈

`take`는 항상 클립보드에서 읽습니다.  
클립보드가 비어 있으면 실패합니다.

---

## 7. 더 깊게 처리하기: `take --deep`, `--llm`

### `--deep` / `--dip`

`take --deep`은 단순 프롬프트 변환이 아니라 **5단계 파이프라인**을 돌립니다.

```bash
prtr take patch --deep
prtr take debug --deep
prtr take refactor --deep
```

단축 별칭:

```bash
prtr take patch --dip
```

파이프라인:

```text
planner -> patcher -> critic -> tester -> reconciler
```

각 단계가 하는 일:

- `planner`: 무엇을 해야 하는지 계획 정리
- `patcher`: 구현 draft와 diff 초안 생성
- `critic`: 위험 요소와 빠진 점 검토
- `tester`: 테스트 케이스와 검증 절차 작성
- `reconciler`: 결과를 하나의 bundle로 통합

### 딥 실행 결과물

repo 안에 다음 경로로 artifact가 남습니다.

```text
.prtr/runs/<run-id>/
```

대표 파일:

- `manifest.json`
- `plan.json`
- `events.jsonl`
- `result/patch_bundle.json`
- `result/patch.diff`
- `result/tests.md`
- `result/summary.md`

### `--llm`

딥 파이프라인의 최종 전달 프롬프트를 AI별 최적 형식으로 바꿉니다.

```bash
prtr take patch --deep --llm=claude
prtr take patch --deep --llm=gemini
prtr take patch --deep --llm=codex
```

형식 차이:

- `claude`: XML 태그 중심
- `gemini`: Markdown 헤더 중심
- `codex`: 번호 목록 + diff block 중심
- 미지정: 범용 Markdown

### 추천 예시

```bash
prtr take patch --deep --llm=claude
prtr take debug --deep --llm=gemini
prtr take refactor --deep --llm=codex
```

---

## 8. 프로젝트 용어 보호: `learn`

프로젝트 고유 용어가 번역 과정에서 깨지지 않게 합니다.

```bash
prtr learn
prtr learn README.md docs
prtr learn --dry-run
prtr learn --reset
```

저장 위치:

```text
.prtr/termbook.toml
```

저장되는 것:

- 어떤 파일을 스캔했는지
- 보호할 용어 목록

예:

```toml
protected_terms = [
  "BuildPrompt",
  "PRTR_TARGET",
  "snake_case",
  "--dry-run",
]
```

이후 `go`와 deep pipeline에서 이 용어들을 번역하지 않고 보존합니다.

---

## 9. 보내기 전에 확인하기: `inspect`

`inspect`는 실제 전송 없이 최종 프롬프트를 확인하는 명령입니다.

```bash
prtr inspect "이 PR 리뷰해줘"
prtr inspect --json "이 에러 분석해줘"
prtr inspect -t codex --template codex-implement -r be "이 함수 개선해줘"
```

자주 쓰는 옵션:

```bash
prtr inspect --explain "이 설정이 어떻게 해석되는지 보여줘"
prtr inspect --diff "번역 전후 차이를 보여줘"
prtr inspect --json "JSON으로 출력해줘"
prtr inspect --lang ja "일본어 출력으로 바꿔줘"
```

중요한 점:

- `inspect`는 전송하지 않습니다.
- 앱 실행도 하지 않습니다.
- 클립보드 복사도 하지 않습니다.

v0.8에서는 `inspect`에서 언어 override는 `--lang`을 권장합니다.  
예전 `--to`는 `inspect`에서는 deprecated이고, `go`/`swap`/`take`에서는 여전히 대상 AI를 의미합니다.

---

## 10. 작업 상태 저장과 복원: `save`, `resume`, `status`, `list`, `prune`

v0.8의 핵심 기능입니다.

### `prtr save`

현재 작업 상태를 **Work Capsule**로 저장합니다.

```bash
prtr save
prtr save "auth refactor midpoint"
prtr save "login retry issue" --note "세션 만료 처리 결정, 테스트는 아직"
```

저장되는 것:

- repo root
- branch
- HEAD SHA
- 마지막 run/session 정보
- TODO
- label, note

저장 위치:

```text
.prtr/capsules/<capsule-id>/
```

### `prtr resume`

저장한 상태를 다시 불러와 이어서 작업합니다.

```bash
prtr resume
prtr resume latest
prtr resume <capsule-id>
prtr resume --to codex
prtr resume <capsule-id> --to gemini --dry-run
```

의미:

- `resume`: 가장 최근 capsule 복구
- `resume --to codex`: 같은 작업 상태를 Codex용 프롬프트로 복원
- `--dry-run`: 실제 전송 없이 resume prompt만 출력

### drift 감지

`resume` 시점에 다음 차이를 감지합니다.

- branch 변경
- HEAD SHA 변경
- working tree 변경

바뀐 내용이 있으면 resume prompt에 drift 경고가 들어갑니다.

### `prtr status`

가장 최근 capsule의 상태를 빠르게 확인합니다.

```bash
prtr status
```

보여주는 것:

- capsule store 경로
- 마지막 저장 시각
- branch drift
- sha drift
- open / done todo 수
- 마지막 target app

### `prtr list`

현재 repo의 capsule 목록을 보여줍니다.

```bash
prtr list
```

### `prtr prune`

오래된 capsule을 지웁니다.

```bash
prtr prune
prtr prune --older-than 30d
prtr prune --dry-run
```

### 자동 저장

v0.8부터는 아래 명령이 성공하면 auto-save가 동작합니다.

- `go`
- `swap`
- `again`
- `take`

즉, 수동으로 `save`하지 않아도 최근 작업 흐름이 capsule로 남습니다.

---

## 11. 셸 감시기: `watch`

`watch`는 셸 활동을 감시해 repo context를 더 빠르게 수집하도록 돕는 기능입니다.

```bash
prtr watch
prtr watch --status
prtr watch --off
```

특징:

- 처음 실행 시 셸 hook 설치를 시도합니다.
- 별도 터미널에서 띄워 두는 foreground daemon에 가깝습니다.
- `--off`로 중지할 수 있습니다.

주의:

- Windows는 지원하지 않습니다.
- zsh 사용자는 `$ZDOTDIR` 경로도 인식합니다.
- help text 기준으로 백그라운드보다는 "별도 터미널에서 돌리는 watcher"로 이해하는 것이 맞습니다.

---

## 12. 설정 파일과 환경 변수

### user config 위치

- `~/.config/prtr/config.toml`
- 또는 `$XDG_CONFIG_HOME/prtr/config.toml`

### project override

- repo 루트의 `.prtr.toml`

### 기본 예시

```toml
deepl_api_key = ""
translation_source_lang = "auto"
translation_target_lang = "en"
default_target = "claude"
default_role = "be"
default_template_preset = "claude-structured"
llm_provider = "claude"

[watch]
enabled = true
notify = false
medium_signals = false

[memory]
enabled = true
auto_save = true
prune_on_write = true
prune_on_resume = true
capsule_retention_days = 30
autosave_retention_days = 14
run_retention_days = 7
max_capsules_per_repo = 200
max_storage_mb_per_repo = 256
store_diff = "stat"
```

설명:

- `translation_source_lang`: 입력 언어
- `translation_target_lang`: 출력 언어
- `default_target`: 기본 AI 앱
- `llm_provider`: deep prompt formatting 기본값
- `watch.*`: watcher 동작
- `memory.*`: capsule 보존 정책

### 환경 변수

```bash
export DEEPL_API_KEY=...
export PRTR_LLM_PROVIDER=claude
export XDG_CONFIG_HOME=$HOME/.config
```

우선순위:

```text
CLI flag > config.toml > env
```

단, `llm_provider`는 구체적으로 아래 순서입니다.

```text
--llm > config.toml llm_provider > PRTR_LLM_PROVIDER
```

---

## 13. 히스토리와 재실행

### 최근 기록 보기

```bash
prtr history
prtr history search review
```

### 특정 기록 다시 실행

```bash
prtr rerun <history-id>
prtr rerun <history-id> --edit
```

### 표시 관리

```bash
prtr pin <history-id>
prtr favorite <history-id>
```

history에는 다음 메타데이터가 저장됩니다.

- 원문 / 번역문 / 최종 프롬프트
- target app
- role / template
- source / target language
- delivery mode
- pasted / submitted 여부
- deep run id / artifact root

---

## 14. 숨겨진 별칭(이스터에그)

v0.8에는 help에 기본 노출되지 않는 culinary alias가 있습니다.

- `dip` -> `take --deep`
- `taste` -> `inspect`
- `plate` -> `swap`
- `marinate` -> `learn`
- `prep` -> `start`

예:

```bash
prtr dip patch
prtr taste "이 프롬프트가 어떻게 조합되는지 보여줘"
prtr plate codex
prtr marinate
prtr prep
```

이 별칭은 재미 요소이지만, 공식 문서나 팀 커뮤니케이션에서는 정식 명령명을 쓰는 편이 좋습니다.

---

## 15. 추천 워크플로

### A. 버그 원인 찾기 -> 패치 만들기

```bash
npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘"
prtr swap gemini
prtr take patch --deep --llm=claude
prtr save "bugfix midpoint"
```

### B. 문서/설계 검토

```bash
prtr go design "이 구조의 확장성 문제를 짚어줘" --to claude
prtr swap codex
prtr take summary
```

### C. 세션 종료 후 이어서 작업

```bash
prtr save "auth refactor paused"
# 터미널 종료
prtr status
prtr resume --to codex
```

### D. 프로젝트 용어 먼저 학습

```bash
prtr learn
prtr go review "이 PR에서 위험한 부분만 짚어줘"
```

---

## 16. 문제 해결

### 먼저 해볼 것

```bash
prtr doctor
prtr doctor --fix
```

### DeepL 키가 없을 때

- 영어 요청은 DeepL 키 없이도 동작합니다.
- 한국어 -> 영어 번역 품질이 필요하면 키를 넣으면 됩니다.
- v0.6.3 이후에는 키가 없어도 hard failure 대신 graceful skip 경로가 있습니다.

### 클립보드 문제

- macOS: `pbcopy`
- Linux: `wl-copy`, `xclip`, `xsel`
- Windows: `clip.exe`

### 대상 앱 실행/붙여넣기 문제

- `claude`, `codex`, `gemini` CLI가 PATH에 있는지 확인
- launcher 설정 확인
- macOS는 접근성 권한 확인
- Linux는 X11/Wayland용 도구 확인

### `take`가 실패할 때

대부분 클립보드가 비어 있어서입니다.

### `resume`이 이상할 때

```bash
prtr status
prtr list
prtr resume --dry-run
```

먼저 drift 경고와 현재 target app이 맞는지 확인하세요.

### `watch`가 안 붙을 때

- zsh라면 `$ZDOTDIR` 위치 확인
- bash는 `~/.bashrc` 또는 `~/.bash_profile` 확인
- `prtr watch --status`로 watcher 상태 확인

---

## 빠른 명령 모음

```bash
# 시작
prtr start
prtr doctor --fix

# 첫 요청
prtr go "이 에러 분석해줘"
prtr go fix "왜 테스트가 깨지는지 찾아줘" --to claude

# 비교/반복
prtr again
prtr swap gemini

# 다음 액션
prtr take patch
prtr take patch --deep --llm=claude

# 용어 보호
prtr learn

# 확인
prtr inspect "이 프롬프트를 확인해줘"

# 상태 저장/복원
prtr save "midpoint"
prtr status
prtr list
prtr resume --to codex
prtr prune --dry-run

# watcher
prtr watch
prtr watch --status
prtr watch --off
```

이 문서 하나만 기억하면 됩니다.

- 처음 질문은 `go`
- 비교는 `swap`
- 후속 액션은 `take`
- 깊은 실행은 `take --deep`
- 용어 보호는 `learn`
- 이어서 작업은 `save / resume`
- 자동 맥락 추적은 `watch`
