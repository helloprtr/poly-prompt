# poly-prompt

[![Project Site](https://img.shields.io/badge/project%20site-live-ff7a1a?style=flat-square)](https://helloprtr.github.io/poly-prompt/)
[![Docs Hub](https://img.shields.io/badge/docs-pages-58f2c5?style=flat-square)](https://helloprtr.github.io/poly-prompt/docs/)
[![Latest Release](https://img.shields.io/badge/release-v1.0.2-1b2c49?style=flat-square)](https://github.com/helloprtr/poly-prompt/releases/tag/v1.0.2)

[English README](README.md) · [한국어 README](README.ko.md) · [문서 허브](https://helloprtr.github.io/poly-prompt/docs/) · [릴리스](https://github.com/helloprtr/poly-prompt/releases)

![prtr banner](images/prtr-banner.png)

**한 줄 소개:** `prtr`는 AI Work Session Manager입니다. 작업 세션을 시작하고, 진행 상황을 저장하고, Claude에서 Gemini나 Codex로 자연스럽게 넘기며, 다시 이어서 작업할 수 있게 해줍니다.

`prtr` 1.0.0의 핵심은 "좋은 첫 프롬프트"보다 "작업이 끊기지 않는 흐름"입니다. 지금 무슨 일을 하는지, 어느 파일을 보고 있는지, 중간에 무엇을 끝냈는지, git 변경이 어디까지 쌓였는지를 세션과 워크 캡슐로 관리해 다음 AI가 바로 이어받을 수 있게 만듭니다.

## 1분 설치

macOS + Homebrew:

```bash
brew tap helloprtr/homebrew-tap
brew install prtr
prtr doctor
```

Linux / Windows:

[GitHub Releases](https://github.com/helloprtr/poly-prompt/releases)에서 플랫폼용 아카이브를 받아 `PATH`에 추가한 뒤 `prtr doctor`를 실행하면 됩니다.

## 초심자용 첫 사용 흐름

```bash
prtr edit internal/session/store.go
prtr checkpoint "store 구조 정리 완료"
prtr @gemini
prtr status
prtr done
```

이 다섯 줄만 이해해도 1.0.0의 핵심은 거의 다 쓴 것입니다.

## 1.0.0 핵심 기능

### 1. 세션

세션은 "지금 이 저장소에서 어떤 작업을 하는가"를 잡아두는 기본 단위입니다.

- `prtr review [files...]`: 리뷰 세션 시작
- `prtr edit [files...]`: 수정 세션 시작
- `prtr fix [files...]`: 버그 수정 세션 시작
- `prtr design [topic]`: 설계 세션 시작
- `prtr`: 현재 세션 이어서 진행하거나 새 세션 시작

세션에는 작업 목표, 파일 범위, 모드, 시작 시점의 git SHA, 체크포인트가 저장됩니다.

### 2. 체크포인트

```bash
prtr checkpoint "로그 파싱 분리 완료"
```

중간 진행 상황을 한 줄 메모로 남기는 기능입니다. 나중에 다른 모델로 핸드오프할 때 이 메모들이 함께 들어가므로, "어디까지 했는지"를 다시 설명할 필요가 크게 줄어듭니다.

### 3. 모델 핸드오프

```bash
prtr @gemini
prtr @codex
```

현재 세션의 목표, 파일 범위, 체크포인트, 시작 이후의 git diff, 마지막 AI 응답까지 엮어서 새 모델로 넘깁니다. 1.0.0에서 가장 실전적인 기능 중 하나입니다.

**자동 응답 캡처 (v1.0.2):** Claude Code, Codex 또는 Gemini CLI를 닫으면 prtr이 자동으로 대화 로그를 읽어 마지막 응답을 `~/.config/prtr/last-response.json`에 저장합니다. 다음 핸드오프 시 별도 복사 없이 자동으로 컨텍스트에 포함됩니다.

- Claude Code: `~/.claude/projects/<repo-slug>/*.jsonl`
- Codex: `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl`
- Gemini CLI: `~/.gemini/tmp/<project-hash>/chats/session-*.json`

### 4. 상태 확인

```bash
prtr status
prtr sessions
```

- `prtr status`: 현재 세션 요약과 최신 Work Capsule 상태를 함께 보여줍니다.
- `prtr sessions`: 현재 저장소의 세션 목록을 보여줍니다.

즉, "지금 무엇을 하고 있는지"와 "최근 저장해둔 이어하기 지점이 무엇인지"를 빠르게 확인할 수 있습니다.

### 5. Work Capsule

세션이 "지금 진행 중인 작업"이라면, Work Capsule은 "나중에 다시 열기 위한 저장점"입니다.

```bash
prtr save "auth 리팩토링 중간 저장"
prtr list
prtr resume
prtr prune --older-than 30d
```

- `save`: 현재 작업 상태를 저장
- `resume`: 저장한 상태를 복원용 프롬프트로 생성
- `list`: 저장점 목록 확인
- `prune`: 오래된 저장점 정리

특히 `resume`은 브랜치, SHA, 변경 파일이 달라졌는지 drift를 감지해서 경고해 줍니다.

### 6. memory / mem / learn

1.0.0에서 초심자가 헷갈리기 쉬운 부분입니다. 별도의 대표 커맨드가 `mem`으로 노출되지는 않지만, 메모리 개념은 두 층으로 존재합니다.

- 작업 메모리: `save`, `resume`, `status`, `list`, `prune`으로 다루는 Work Capsule
- 저장소 메모리: `prtr learn`으로 쌓는 repo memory

```bash
prtr learn
prtr learn README.md docs --dry-run
```

`learn`은 프로젝트 용어, 파일명, 도메인 표현을 보호해 번역이나 프롬프트 변형 중에도 중요한 이름이 망가지지 않게 돕습니다.

### 7. 예전 loop 명령도 여전히 사용 가능

아래 명령들은 1.0.0에서 기본 전면은 아니지만 아직 작동합니다.

- `prtr go`
- `prtr swap`
- `prtr take`
- `prtr again`
- `prtr inspect`
- `prtr learn`

즉, 예전 loop 사용자가 바로 깨지지는 않지만, 새 사용자에게는 세션 중심 흐름을 먼저 익히는 편이 훨씬 쉽습니다.

## 커맨드 한눈에 보기

| 커맨드 | 역할 |
|---|---|
| `prtr` | 현재 세션 이어서 진행 또는 새 세션 시작 |
| `prtr review [files]` | 코드 리뷰 세션 |
| `prtr edit [files]` | 코드 수정 세션 |
| `prtr fix [files]` | 버그 수정 세션 |
| `prtr design [topic]` | 설계 세션 |
| `prtr checkpoint "메모"` | 진행 상황 저장 |
| `prtr @gemini` / `@codex` | 현재 세션을 다른 모델로 핸드오프 |
| `prtr done` | 세션 완료 처리 |
| `prtr sessions` | 현재 저장소 세션 목록 |
| `prtr status` | 현재 세션 + 최신 Work Capsule 상태 |
| `prtr save [label]` | Work Capsule 저장 |
| `prtr resume [id]` | Work Capsule 복원 |
| `prtr list` | Capsule 목록 |
| `prtr prune` | 오래된 Capsule 정리 |
| `prtr learn [paths...]` | 저장소 용어 메모리 구축 |
| `prtr doctor` | 환경 점검 |
| `prtr setup` | 초기 설정 |

## 문서

- 시작점: [문서 허브](https://helloprtr.github.io/poly-prompt/docs/)
- 설치와 업데이트: [INSTALLATION.md](INSTALLATION.md)
- 한국어 상세 가이드: [docs/guide.ko.md](docs/guide.ko.md)
- 영어 상세 가이드: [docs/guide.md](docs/guide.md)
- 커맨드 레퍼런스: [docs/reference.md](docs/reference.md)
- 프로젝트 사이트: [helloprtr.github.io/poly-prompt](https://helloprtr.github.io/poly-prompt/)

## 기여하기

- 기여 가이드: [CONTRIBUTING.md](CONTRIBUTING.md)
- 입문용 이슈: [good first issue](https://github.com/helloprtr/poly-prompt/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22)
- 추가 도움 요청: [help wanted](https://github.com/helloprtr/poly-prompt/issues?q=is%3Aopen+is%3Aissue+label%3A%22help+wanted%22)
- 아이디어와 워크플로 피드백: [Discussions](https://github.com/helloprtr/poly-prompt/discussions)
