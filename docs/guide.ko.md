# prtr 사용자 가이드 (한국어)

`prtr`은 터미널과 AI 도구(Claude / Codex / Gemini) 사이에 위치하는 **AI 작업 커맨드 레이어**입니다.

로그, 다프, 의도를 입력하면 prtr이 각 AI에 최적화된 프롬프트를 조합해 클립보드에 담고, 앱을 열어 붙여넣기까지 처리합니다. 여러 AI를 비교하거나, 답변을 바로 다음 액션으로 이어가거나, 반복 작업을 자동화하는 데 사용합니다.

---

## 목차

1. [prtr이란?](#1-prtr이란)
2. [설치](#2-설치)
3. [첫 5분](#3-첫-5분)
4. [첫 번째 요청: prtr go](#4-첫-번째-요청-prtr-go)
5. [비교하기: prtr swap / again](#5-비교하기-prtr-swap--again)
6. [답변을 다음 액션으로: prtr take](#6-답변을-다음-액션으로-prtr-take)
   - [6.1 --deep 모드](#61---deep-모드)
   - [6.2 --llm 프로바이더 설정](#62---llm-프로바이더-설정)
7. [용어 보호: prtr learn](#7-용어-보호-prtr-learn)
8. [라우팅 확인: prtr inspect](#8-라우팅-확인-prtr-inspect)
9. [설정 파일](#9-설정-파일)
10. [히스토리](#10-히스토리)
11. [문제 해결](#11-문제-해결)

---

## 1. prtr이란?

`prtr`(프리터)은 개발자가 AI 도구를 더 효율적으로 사용할 수 있도록 설계된 **커맨드 레이어 CLI**입니다.

### 왜 필요한가?

AI 도구를 일상적으로 사용하다 보면 반복되는 불편함이 생깁니다.

- 같은 컨텍스트를 Claude에도, Gemini에도 직접 붙여넣어야 한다
- AI가 돌려준 답변을 받아 "그러면 이걸 패치로 만들어줘" 같은 후속 요청을 다시 수동으로 작성해야 한다
- 프로젝트 고유 용어가 번역 과정에서 깨진다
- 어떤 AI가 이 유형의 질문에 더 잘 답하는지 비교하기 번거롭다

`prtr`은 이 흐름 전체를 자동화합니다.

### 핵심 루프

```
의도 입력 → prtr이 프롬프트 조합 → AI 앱에 전달 → 답변을 다음 액션으로 연결
```

### 지원 AI

| 대상    | 키워드    |
|---------|-----------|
| Claude  | `claude`  |
| OpenAI Codex | `codex` |
| Google Gemini | `gemini` |

---

## 2. 설치

### macOS — Homebrew (권장)

```bash
brew tap koouu/prtr
brew install prtr
```

### Linux — 직접 설치

```bash
curl -sSL https://github.com/koouu/prtr/releases/latest/download/prtr-linux-amd64 -o prtr
chmod +x prtr
sudo mv prtr /usr/local/bin/
```

ARM64(예: Raspberry Pi, Apple Silicon Linux VM)인 경우:

```bash
curl -sSL https://github.com/koouu/prtr/releases/latest/download/prtr-linux-arm64 -o prtr
chmod +x prtr
sudo mv prtr /usr/local/bin/
```

### Windows — PowerShell

```powershell
Invoke-WebRequest -Uri "https://github.com/koouu/prtr/releases/latest/download/prtr-windows-amd64.exe" -OutFile "prtr.exe"
Move-Item prtr.exe "$env:USERPROFILE\bin\prtr.exe"
```

`$env:USERPROFILE\bin`이 PATH에 없다면 추가하세요.

### 소스 빌드

Go 1.21 이상이 필요합니다.

```bash
git clone https://github.com/koouu/prtr.git
cd prtr
go build -o prtr ./cmd/prtr
sudo mv prtr /usr/local/bin/
```

### 설치 확인

```bash
prtr version
```

---

## 3. 첫 5분

설치 직후 바로 시작할 수 있습니다. API 키나 계정 없이도 동작하는 경로부터 확인해 보세요.

### 데모 실행

```bash
prtr demo
```

실제 전송 없이 prtr이 어떤 프롬프트를 만드는지 보여줍니다.

### dry-run으로 미리 보기

```bash
prtr go "이 에러 원인 분석해줘" --dry-run
```

`--dry-run`은 클립보드 복사나 앱 실행 없이 조합된 프롬프트만 터미널에 출력합니다.

### 가이드 방식으로 시작

처음이라면 대화형 설정 흐름을 사용할 수 있습니다:

```bash
prtr start
```

`start`는 다음을 처리합니다:
- 필요한 경우 최소한의 온보딩 안내
- 첫 전송 전 `doctor` 진단 실행
- 요청 문구를 따로 전달하지 않아도 프롬프트 입력 안내

언어 설정까지 포함한 전체 설정을 원하면:

```bash
prtr setup
```

시스템 상태 진단은 언제든지:

```bash
prtr doctor
prtr doctor --fix   # 자동 수정 시도
```

---

## 4. 첫 번째 요청: prtr go

`prtr go`는 가장 기본이 되는 명령입니다. 한국어로 의도를 입력하면 prtr이 영어 프롬프트로 변환한 뒤 지정한 AI 앱에 전달합니다.

### 기본 전송

```bash
prtr go "이 에러 원인 분석해줘"
```

### 모드 선택

요청 유형에 맞는 모드를 선택하면 프롬프트 품질이 높아집니다.

```bash
prtr go fix "왜 테스트가 깨지는지 정확한 원인만 찾아줘"
prtr go review "이 PR에서 위험한 부분만 짚어줘"
prtr go design "이 기능 구조 설계해줘"
```

| 모드      | 적합한 상황                          |
|-----------|-------------------------------------|
| `fix`     | 버그 분석, 오류 원인 추적            |
| `review`  | 코드 리뷰, 위험도 평가               |
| `design`  | 아키텍처 설계, 구조 논의             |

### AI 앱 선택

```bash
prtr go "이 구조 문제점 봐줘" --to claude
prtr go fix "왜 테스트가 깨지는지 봐줘" --to codex
prtr go review "이 설계 위험도 평가해줘" --to gemini
```

`--to` 없이 실행하면 설정 파일의 기본값 또는 `claude`가 사용됩니다.

### 전송 전 편집

```bash
prtr go design "이 기능 구조 설계해줘" --to gemini --edit
```

`--edit`는 프롬프트가 클립보드로 가기 전에 에디터를 열어 직접 수정할 수 있게 합니다.

### 로그/파일 파이핑

prtr은 stdin 파이프를 증거(evidence) 데이터로 처리합니다.

```bash
# 테스트 실패 로그와 함께 요청
npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘"

# 크래시 로그만 전달 (의도 문구 없이)
cat crash.log | prtr go fix

# 특정 파일 내용을 컨텍스트로 추가
cat src/api.go | prtr go review "이 파일 구조가 안전한지 봐줘"
```

입력 처리 규칙:
- 메시지와 파이프 입력을 함께 주면 → 파이프 데이터가 증거로 첨부됩니다
- 파이프 입력만 주면 → 파이프 데이터가 요청 본문이 됩니다
- Git 저장소 내에서 실행하면 → 저장소 이름, 현재 브랜치, 변경 파일 요약이 자동으로 포함됩니다
- `--no-context`를 사용하면 자동 컨텍스트와 증거 첨부가 비활성화됩니다

---

## 5. 비교하기: prtr swap / again

### prtr again — 같은 요청 반복

직전 흐름을 그대로 다시 실행합니다:

```bash
prtr again
prtr again --edit       # 프롬프트 수정 후 재실행
prtr again --dry-run    # 미리 보기만
```

### prtr swap — 다른 AI로 재전송

직전 요청을 다른 AI 앱으로 보냅니다. 컨텍스트를 새로 구성할 필요 없이 비교가 가능합니다.

```bash
prtr swap claude
prtr swap codex
prtr swap gemini
prtr swap gemini --edit
prtr swap claude --dry-run
```

`swap`은 직전 요청과 모드를 유지하되, 대상 앱의 템플릿에 맞게 프롬프트를 재조합합니다. 단순히 이전 앱의 프롬프트를 그대로 복사하지 않습니다.

#### 실전 예시: Claude vs Gemini 비교

```bash
# Claude에 먼저 보내고
prtr go review "이 PR 병합해도 안전한지 판단해줘" --to claude

# Gemini에도 같은 요청 보내기
prtr swap gemini
```

---

## 6. 답변을 다음 액션으로: prtr take

AI가 돌려준 답변을 클립보드에 복사한 상태에서 `prtr take`를 실행하면, 그 내용을 읽어 다음 작업에 맞는 새 프롬프트를 생성합니다.

```bash
prtr take patch
prtr take test --to codex
prtr take commit --dry-run
prtr take summary --edit
```

### 지원 액션

| 액션       | 동작                                           |
|------------|------------------------------------------------|
| `patch`    | 답변 내용을 코드 패치 요청으로 변환            |
| `test`     | 답변 기반으로 테스트 작성 요청 생성            |
| `commit`   | 변경 내용을 커밋 메시지로 요약 요청            |
| `summary`  | 답변 내용을 간결하게 요약 요청                 |
| `clarify`  | 모호한 부분을 명확히 해달라는 후속 요청        |
| `issue`    | 버그 또는 개선 사항을 이슈 형식으로 작성 요청  |
| `plan`     | 답변을 기반으로 실행 계획 생성 요청            |
| `debug`    | 클립보드 내용을 디버깅 분석 요청으로 변환      |
| `refactor` | 코드를 리팩터링 요청 프롬프트로 변환           |

`take`는 항상 클립보드에서 읽고, 선택한 액션에 맞는 영어 요청을 새로 생성한 뒤 `go`와 동일한 전달 흐름으로 처리합니다.

### 6.1 `--deep` 모드

`--deep` (또는 단축 플래그 `--dip`)는 **딥 분석 파이프라인**을 활성화합니다.

단순 프롬프트 전달 대신 5개 워커가 순서대로 작업을 처리합니다:

```
planner → patcher → critic → tester → reconciler
```

| 워커          | 역할                                              |
|---------------|--------------------------------------------------|
| `planner`     | 작업 범위를 분석하고 실행 계획을 수립             |
| `patcher`     | 실제 코드 변경 또는 패치 생성                     |
| `critic`      | 생성된 변경 사항을 검토하고 약점 식별             |
| `tester`      | 테스트 파일 탐지 및 테스트 커버리지 제안          |
| `reconciler`  | 충돌을 해소하고 최종 결과물을 통합                |

복잡한 변경, 리팩터링, 버그 수정처럼 단순 답변 이상의 심층 분석이 필요한 작업에 적합합니다.

```bash
prtr take patch --deep
prtr take debug --deep
prtr take refactor --deep
```

`--dip`은 `--deep`의 단축 별칭입니다:

```bash
prtr take patch --dip
```

### 6.2 `--llm` 프로바이더 설정

`--llm` 플래그는 **프로바이더 최적화 전달 프롬프트**를 생성합니다. 각 AI가 선호하는 형식에 맞게 프롬프트 구조를 자동으로 조정합니다.

```bash
prtr take patch --llm=claude
prtr take patch --llm=gemini
prtr take patch --llm=codex
```

| 프로바이더 | 프롬프트 형식                    |
|------------|----------------------------------|
| `claude`   | XML 태그 구조 (`<task>`, `<context>` 등) |
| `gemini`   | Markdown 헤더 (`##`, `###` 등)   |
| `codex`    | 번호 목록 형식                   |

`--deep`과 함께 사용하면 딥 파이프라인의 출력도 해당 형식에 맞게 구성됩니다:

```bash
prtr take patch --deep --llm=claude
prtr take patch --dip --llm=claude
prtr take debug --deep --llm=gemini
prtr take refactor --deep --llm=codex
```

#### 실전 예시

AI가 "이 함수를 리팩터링하면 좋겠다"는 제안을 돌려줬다면:

1. 답변을 클립보드에 복사
2. 딥 분석과 Claude 최적화 프롬프트로 실제 패치 요청 생성:

```bash
prtr take patch --deep --llm=claude
```

---

## 7. 용어 보호: prtr learn

프로젝트 고유 용어(함수 이름, 플래그, 약어 등)가 번역 과정에서 바뀌는 것을 방지합니다.

### 사용법

```bash
# 저장소 전체에서 용어 자동 수집
prtr learn

# 특정 파일/디렉토리 지정
prtr learn README.md docs

# 수집될 용어 미리 보기 (실제 저장 안 함)
prtr learn --dry-run

# 저장된 termbook 초기화
prtr learn --reset
```

### 저장 내용

`learn`은 다음을 `.prtr/termbook.toml`에 저장합니다:

- 용어 수집에 사용된 소스 파일 목록
- 보호 대상 용어 (예: `BuildPrompt`, `PRTR_TARGET`, `snake_case`, `--dry-run`)

이후 `prtr go`를 실행할 때마다 termbook이 자동으로 로드되어 해당 용어가 번역 대상에서 제외됩니다. `--no-context` 플래그를 사용하면 이 동작을 비활성화할 수 있습니다.

### 예시 termbook

```toml
# .prtr/termbook.toml
protected_terms = [
  "BuildPrompt",
  "PRTR_TARGET",
  "snake_case",
  "--dry-run",
  "termbook",
  "MyService",
]
```

---

## 8. 라우팅 확인: prtr inspect

실제 전송 없이 프롬프트가 어떻게 조합되는지 확인합니다. 설정 디버깅이나 프롬프트 품질 확인에 유용합니다.

```bash
prtr inspect "이 PR 리뷰해줘"
prtr inspect --json "이 에러 분석해줘"
prtr inspect -t codex --template codex-implement -r be "이 함수 개선해줘"
```

### 유용한 플래그

```bash
# 설정이 어떻게 해석되는지 설명
prtr inspect --explain "이 설정이 어떻게 해석되는지 보여줘"

# 번역 전후 차이 확인
prtr inspect --diff "이 문장이 번역 후 어떻게 바뀌는지 보여줘"

# JSON 형식으로 출력
prtr inspect --json "JSON으로 결과를 받고 싶어"
```

`inspect`는 미리 보기 전용입니다:
- 클립보드 복사 없음
- 앱 실행 없음
- 붙여넣기 없음

---

## 9. 설정 파일

`prtr`의 전역 설정은 `~/.config/prtr/config.toml`에 저장됩니다.

### 파일 위치

| 플랫폼  | 경로                                  |
|---------|---------------------------------------|
| macOS   | `~/.config/prtr/config.toml`          |
| Linux   | `~/.config/prtr/config.toml`          |
| Windows | `%APPDATA%\prtr\config.toml`          |

### 주요 설정

```toml
# ~/.config/prtr/config.toml

# LLM 프로바이더 기본값 (claude / gemini / codex)
llm_provider = "claude"

# LLM API 키
llm_api_key  = "sk-ant-..."

# 기본 DeepL API 키 (번역 기능 사용 시)
deepl_api_key = "your-deepl-key"

# 기본 입력 언어
default_input_lang = "ko"

# 기본 출력 언어
default_output_lang = "en"

# 기본 대상 AI
default_target = "claude"

# 기본 역할
default_role = "be"
```

`llm_provider`를 설정하면 `--llm` 플래그를 매번 입력하지 않아도 됩니다. `prtr take patch --deep`만 실행해도 설정된 프로바이더 형식으로 프롬프트가 생성됩니다.

### 환경 변수

설정 파일 대신 환경 변수로도 지정할 수 있습니다:

```bash
export PRTR_LLM_PROVIDER=claude
export PRTR_LLM_API_KEY=sk-ant-...
```

환경 변수는 설정 파일보다 우선합니다.

### 프로젝트별 설정

저장소 루트에 `.prtr.toml`을 두면 해당 프로젝트에서만 적용되는 설정을 지정할 수 있습니다:

```toml
# .prtr.toml (저장소 루트)

[shortcuts.review]
target = "claude"
role = "review"
template_preset = "claude-review"
translation_target_lang = "en"

[profiles.incident]
target = "gemini"
role = "be"
template_preset = "gemini-stepwise"
context = "장애 대응, 롤백, 고객 영향도에 집중해줘."
```

---

## 10. 히스토리

모든 `go` 실행은 로컬에 기록됩니다.

### 기본 명령

```bash
# 최근 실행 목록 보기
prtr history

# 키워드로 검색
prtr history search review

# 특정 항목 다시 실행
prtr rerun <history-id>

# 수정 후 재실행
prtr rerun <history-id> --edit
```

### 즐겨찾기와 핀

자주 쓰는 항목을 표시해 두면 쉽게 재사용할 수 있습니다:

```bash
prtr pin <history-id>
prtr favorite <history-id>
```

### 저장 메타데이터

각 항목에는 다음이 기록됩니다:
- 소스/대상 언어
- 번역 모드와 결정 내용
- 실행 대상 AI
- 전달 모드
- 붙여넣기/제출 여부

---

## 11. 문제 해결

### 기본 진단

무엇이든 잘 안 된다면 먼저 실행:

```bash
prtr doctor
prtr doctor --fix
```

`--fix`는 설정 파일 재생성 등 안전하게 자동 처리할 수 있는 문제를 수정하고, 그 외는 수동 조치 방법을 안내합니다.

### DeepL 키 없음

```bash
prtr doctor   # 상태 확인
prtr setup    # 키 입력 안내
```

영어 요청은 DeepL 키 없이도 동작합니다. 한국어 입력 번역이 필요할 때만 키가 필요합니다.

### 클립보드 오류

| 플랫폼  | 확인 사항                                    |
|---------|---------------------------------------------|
| macOS   | `pbcopy` 사용 가능 여부 확인                |
| Linux   | `wl-copy`, `xclip`, 또는 `xsel` 설치 여부  |
| Windows | `clip.exe` 사용 가능 여부 확인              |

### 앱 실행/붙여넣기 오류

- 대상 CLI(claude, codex, gemini)가 설치되어 PATH에 있는지 확인
- **macOS**: Terminal.app 설치 여부 및 손쉬운 사용(Accessibility) 권한 부여 여부 확인
- **Linux**: 그래픽 세션 실행 중인지, `xdotool`(X11) 또는 `wtype`(Wayland) 설치 여부 확인
- **Windows**: 대화형 데스크톱 세션인지, `powershell.exe` 또는 `pwsh.exe` 사용 가능한지 확인
- `prtr doctor` 재실행

### 번역이 이상할 때

```bash
# 번역 전후 차이 확인
prtr inspect --diff "번역 테스트 문장"

# 강제 번역
prtr go "짧은 명령도 번역해줘" --translation-mode force

# 번역 건너뛰기
prtr go "Keep this prompt exactly as written" --translation-mode skip
```

### --deep 파이프라인이 느릴 때

`--deep` 모드는 5개 워커가 순차적으로 실행되므로 단순 `take`보다 시간이 걸립니다. 복잡한 작업에만 사용하고, 빠른 확인이 필요할 때는 `--deep` 없이 실행하세요.

### LLM API 키 오류

```bash
# 설정 파일 확인
cat ~/.config/prtr/config.toml

# 환경 변수 확인
echo $PRTR_LLM_PROVIDER
echo $PRTR_LLM_API_KEY
```

API 키가 올바른지, 프로바이더 이름이 `claude` / `gemini` / `codex` 중 하나인지 확인하세요.

---

## 부록: 자주 쓰는 명령 모음

```bash
# 기본 흐름
prtr go "이 에러 원인 분석해줘"
prtr go fix "테스트가 왜 깨지는지 찾아줘" --to claude
prtr swap gemini

# 딥 분석
prtr take patch --deep
prtr take patch --deep --llm=claude
prtr take debug --deep --llm=gemini
prtr take refactor --deep

# 로그 파이핑
npm test 2>&1 | prtr go fix "왜 깨지는지 찾아줘"
cat crash.log | prtr go fix

# 미리 보기
prtr go "이 구조 분석해줘" --dry-run
prtr inspect "이 PR 리뷰해줘"

# 용어 보호
prtr learn
prtr learn README.md docs

# 히스토리
prtr history
prtr rerun <id>
```
