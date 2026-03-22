# poly-prompt

[![Project Site](https://img.shields.io/badge/project%20site-live-ff7a1a?style=flat-square)](https://helloprtr.github.io/poly-prompt/)
[![Docs Hub](https://img.shields.io/badge/docs-pages-58f2c5?style=flat-square)](https://helloprtr.github.io/poly-prompt/docs/)
[![Latest Release](https://img.shields.io/badge/release-v1.0.0-1b2c49?style=flat-square)](https://github.com/helloprtr/poly-prompt/releases/tag/v1.0.0)

[English README](README.md) · [한국어 README](README.ko.md) · [문서 허브](https://helloprtr.github.io/poly-prompt/docs/) · [릴리스](https://github.com/helloprtr/poly-prompt/releases)

![prtr banner](images/prtr-banner.png)

**한 줄 소개:** `prtr`는 AI 작업 세션 매니저입니다. 집중 세션을 시작하고, Claude에게 드라이브를 맡기고, 진행 상황을 체크포인트로 저장하고, Gemini나 Codex로 깔끔하게 넘깁니다.

`prtr`는 AI 루프 전반에서 지금 무엇을 작업 중인지 추적합니다. 세션을 시작하고, 진행 상황을 체크포인트로 남기고, 다른 모델에 넘기세요 — 컨텍스트를 손으로 다시 조립하지 않아도 됩니다.

## 1분 Quick Start

macOS + Homebrew:

```bash
brew tap helloprtr/homebrew-tap
brew install prtr
prtr review internal/app/app.go
```

Linux / Windows:

[GitHub Releases](https://github.com/helloprtr/poly-prompt/releases)에서 플랫폼용 아카이브를 내려받아 `PATH`에 추가한 뒤, 위의 마지막 줄을 그대로 실행하면 됩니다.

실제 세션은 이렇게 시작합니다:

```bash
prtr edit internal/session/store.go   # 집중 수정 세션 시작
prtr checkpoint "store 리팩토링 완료"  # 진행 상황 저장
prtr @gemini                           # Gemini로 핸드오프
prtr done                              # 세션 완료 처리
```

`--dry-run` 흐름은 API 키 없이 바로 확인할 수 있습니다.

## 세션 커맨드

| 커맨드 | 역할 |
|---|---|
| `prtr` | 진행 중인 세션 재개 또는 새 세션 시작 |
| `prtr review [파일]` | 코드 리뷰 세션 시작 |
| `prtr edit [파일]` | 집중 수정 세션 시작 |
| `prtr fix [파일]` | 버그 수정 세션 시작 |
| `prtr design [주제]` | 설계 세션 시작 |
| `prtr @gemini` / `@codex` | 현재 세션을 다른 모델로 핸드오프 |
| `prtr checkpoint "메모"` | 진행 상황 메모 저장 |
| `prtr done` | 세션 완료 처리 |
| `prtr sessions` | 현재 리포의 세션 목록 |
| `prtr status` | 현재 세션 상태 및 git diff 요약 확인 |

## 문서

- 시작점: [문서 허브](https://helloprtr.github.io/poly-prompt/docs/)
- 설치와 업데이트: [INSTALLATION.md](INSTALLATION.md)
- 한국어 사용 가이드: [docs/guide.ko.md](docs/guide.ko.md)
- 영어 사용 가이드: [docs/guide.md](docs/guide.md)
- 커맨드 레퍼런스: [docs/reference.md](docs/reference.md)
- 프로젝트 사이트: [helloprtr.github.io/poly-prompt](https://helloprtr.github.io/poly-prompt/)

## 기여하기

- 기여 가이드: [CONTRIBUTING.md](CONTRIBUTING.md)
- 입문용 이슈: [good first issue](https://github.com/helloprtr/poly-prompt/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22)
- 조금 더 넓은 도움 요청: [help wanted](https://github.com/helloprtr/poly-prompt/issues?q=is%3Aopen+is%3Aissue+label%3A%22help+wanted%22)
- 아이디어와 워크플로 피드백: [Discussions](https://github.com/helloprtr/poly-prompt/discussions)

이 저장소를 운영한다면 `good first issue`는 3~5개 정도만 작고 최신 상태로 유지하는 편이 좋습니다. 좋은 스타터 이슈는 사용자 문제를 짧고 분명하게 설명하고, 완료 기준과 로컬 검증 명령을 함께 적어 둡니다.
