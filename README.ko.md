# poly-prompt

[![Project Site](https://img.shields.io/badge/project%20site-live-ff7a1a?style=flat-square)](https://helloprtr.github.io/poly-prompt/)
[![Docs Hub](https://img.shields.io/badge/docs-pages-58f2c5?style=flat-square)](https://helloprtr.github.io/poly-prompt/docs/)
[![Latest Release](https://img.shields.io/badge/release-v0.6.2-1b2c49?style=flat-square)](https://github.com/helloprtr/poly-prompt/releases/tag/v0.6.2)

**prtr는 AI 작업을 위한 커맨드 레이어입니다.**

AI 코딩 도구를 쓰는 개발자가 같은 의도를 매번 다시 쓰지 않아도 되게 해주는 커맨드 레이어.
로그, diff, 한국어 의도를 받아서 Claude, Codex, Gemini로 바로 전달합니다.

## 30초 루프

```bash
npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘"
prtr swap gemini
prtr take patch --deep
prtr learn
```

하나의 의도. 다른 AI 앱. 구조화된 아티팩트. 리포 메모리.

## 설치

#### Homebrew (macOS)

```bash
brew tap helloprtr/homebrew-tap
brew install prtr
```

#### GitHub Releases (Linux / Windows)

플랫폼에 맞는 아카이브를 [릴리즈 페이지](https://github.com/helloprtr/poly-prompt/releases)에서 다운로드하세요.

- Linux: `prtr_<version>_linux_amd64.tar.gz` 또는 `prtr_<version>_linux_arm64.tar.gz`
- Windows: `prtr_<version>_windows_amd64.zip` 또는 `prtr_<version>_windows_arm64.zip`

#### 소스에서 빌드

```bash
git clone https://github.com/helloprtr/poly-prompt.git
cd poly-prompt
go build ./cmd/prtr
```

## 처음 시작하기

API 키 없이 바로 동작을 확인할 수 있습니다:

```bash
prtr demo                          # 안전한 미리보기 — 앱 실행 없음, 키 불필요
prtr go "이 에러 설명해줘" --dry-run
```

전체 루프를 쓰려면 설정을 마칩니다:

```bash
prtr setup     # DeepL 키 저장, 기본 앱 설정
prtr doctor    # 준비된 항목과 선택 항목 확인
```

설정 후 바로 쓸 수 있는 예시:

```bash
prtr go fix "왜 테스트가 깨지는지 원인만 찾아줘"
prtr take patch --deep             # 리스크 분석 + 테스트 계획 + 딜리버리 프롬프트 자동 생성
```

## 핵심 커맨드

| 커맨드 | 역할 |
|---|---|
| `go` | 의도를 번역하고, 컨텍스트를 추가하고, AI 앱을 열어 프롬프트를 붙여넣기 |
| `swap` | 마지막 실행을 다른 AI 앱으로 재전송 |
| `take` | AI 출력을 다음 구조화된 액션으로 전환 |
| `again` | 마지막 실행 재실행; `--edit`으로 보내기 전에 수정 가능 |
| `learn` | 리포 메모리와 보호 용어 목록 구축 |
| `inspect` | 미리보기 전용 전문가 경로: JSON, explain, diff |
| `take --deep` | 내부 파이프라인 실행: 리스크 분석, 테스트 계획, 딜리버리 프롬프트 |

## `take --deep`

`--deep`은 AI에 전달하기 전에 내부 멀티스텝 파이프라인을 실행합니다:

1. **분석** — diff, 로그, 리포 컨텍스트 읽기
2. **리스크** — 깨질 수 있는 부분 식별
3. **테스트 계획** — 검증 단계 자동 생성
4. **딜리버리 프롬프트** — 대상 AI 프로바이더에 맞는 포맷으로 출력

```bash
# AI가 코드를 분석하고 리스크, 테스트 계획, 딜리버리 프롬프트를 자동 생성
prtr take patch --deep
prtr take test --deep          # 테스트 작성 플로우
prtr take debug --deep         # 디버그 플로우
prtr take refactor --deep      # 리팩터 플로우

# 프로바이더별 최적화 포맷 (XML/Markdown/번호 목록)
prtr take patch --deep --llm=claude
prtr take patch --deep --llm=gemini
prtr take patch --deep --llm=codex
```

### 프로바이더별 포맷

AI 앱마다 기대하는 프롬프트 구조가 다릅니다. `--llm`으로 딜리버리 프롬프트 포맷을 지정합니다:

| 프로바이더 | 포맷 |
|---|---|
| `claude` | XML 태그 (`<task>`, `<context>`, `<constraints>`) |
| `gemini` | Markdown 섹션 |
| `codex` | 번호 목록 방식 지시문 |

### 기본 프로바이더 설정

매번 `--llm`을 붙이지 않으려면 config에 지정합니다:

```toml
# ~/.config/prtr/config.toml
llm_provider = "claude"   # 이후엔 --llm 없이도 Claude XML 포맷 자동 적용
```

환경 변수로도 설정할 수 있습니다:

```bash
export PRTR_LLM_PROVIDER=claude
```

설정하면 `--deep` 실행 시 자동으로 해당 포맷이 적용됩니다.

## 루프 유지하기

`prtr`의 핵심은 한 번 보내는 것이 아니라 다음 액션을 더 빠르게 만드는 것입니다.

### `prtr again`

마지막 실행을 같은 모드와 앱으로 재실행합니다.

```bash
prtr again
prtr again --edit    # 보내기 전에 프롬프트 수정
```

### `prtr swap`

마지막 실행을 다른 앱으로 재전송합니다. 컨텍스트를 다시 만들 필요가 없습니다.

```bash
prtr swap claude
prtr swap gemini
prtr swap codex
```

앱을 바꿀 때 `prtr`은 핵심 프롬프트는 유지하고 대상 앱에 맞는 프롬프트 구조로 재해석합니다.

### `prtr take`

AI 출력을 클립보드에서 읽어 다음 액션으로 변환합니다. 수동으로 다시 쓸 필요가 없습니다.

```bash
prtr take patch
prtr take test --to codex
prtr take clarify --from last-prompt
prtr take issue --dry-run
prtr take plan --edit
```

`take`는 `patch`, `test`, `commit`, `summary`, `clarify`, `issue`, `plan`을 지원합니다.

### `prtr learn`

번역을 거쳐도 살아남아야 하는 프로젝트 용어와 리포 메모리를 학습시킵니다.

```bash
prtr learn
prtr learn README.md docs
prtr learn --dry-run
prtr learn --reset
```

`learn`은 리포 로컬 `.prtr/termbook.toml`과 `.prtr/memory.toml`을 생성합니다.
이후 `go`, `take`, `inspect`가 보호 용어, 리포 요약, 프롬프트 가이드를 자동으로 재사용합니다.

## 자주 쓰는 플래그

`go`, `again`, `swap`, `take`의 기본 전송 플래그:

- `--to <app>`: `claude`, `codex`, `gemini` 중 선택
- `--edit`: 전달 전에 최종 프롬프트를 검토하고 수정
- `--dry-run`: 미리보기 전용, 실행 또는 붙여넣기 없음
- `--no-context`: 프롬프트 텍스트가 있을 때 파이프된 stdin 증거 무시

`take --deep` 전용 플래그:

- `--llm=<provider>`: `claude`, `gemini`, `codex` 중 하나로 딜리버리 포맷 지정

## `inspect` 모드

전문가 경로. 미리보기 전용으로 클립보드 복사, 앱 실행, 붙여넣기를 하지 않습니다.

```bash
prtr inspect "이 에러 원인 분석해줘"
prtr inspect --json "이 PR 리뷰해줘"
prtr inspect -t codex --template codex-implement -r be "이 함수 개선해줘"
```

## 링크

- 전체 설치 및 업데이트 가이드: [INSTALLATION.md](INSTALLATION.md)
- 커맨드 레퍼런스 (영어): [docs/guide.md](docs/guide.md)
- 커맨드 레퍼런스 (한국어): [docs/guide.ko.md](docs/guide.ko.md)
- 프로젝트 사이트: [helloprtr.github.io/poly-prompt](https://helloprtr.github.io/poly-prompt/)
