# poly-prompt

[![Project Site](https://img.shields.io/badge/project%20site-live-ff7a1a?style=flat-square)](https://helloprtr.github.io/poly-prompt/)
[![Docs Hub](https://img.shields.io/badge/docs-pages-58f2c5?style=flat-square)](https://helloprtr.github.io/poly-prompt/docs/)
[![Latest Release](https://img.shields.io/badge/release-v0.7.0-1b2c49?style=flat-square)](https://github.com/helloprtr/poly-prompt/releases/tag/v0.7.0)

[English README](README.md) · [한국어 README](README.ko.md) · [문서 허브](https://helloprtr.github.io/poly-prompt/docs/) · [릴리스](https://github.com/helloprtr/poly-prompt/releases)

![prtr banner](images/prtr-banner.png)

**한 줄 소개:** `prtr`는 의도, 로그, diff를 Claude, Codex, Gemini용 다음 AI 액션으로 바꿔주는 커맨드 레이어입니다.

`prtr`는 첫 프롬프트를 더 빨리 보내게 해줄 뿐 아니라, 그다음 루프까지 이어지게 해줍니다. 같은 컨텍스트를 다시 조립하지 않고 한 번 라우팅한 뒤, 다른 AI 앱과 비교하고, 답변을 다음 구조화된 액션으로 넘길 수 있습니다.

## 동작 화면

![prtr go, swap, take 루프를 보여주는 애니메이션 데모](images/prtr-routing-history.gif)

한국어 요청 한 번. 다른 AI 앱 비교 한 번. 다음 액션 생성 한 번. 프롬프트를 다시 붙이지 않아도 됩니다.

## 1분 Quick Start

macOS + Homebrew:

```bash
brew tap helloprtr/homebrew-tap
brew install prtr
prtr demo
prtr go "explain this error" --dry-run
```

Linux / Windows:

[GitHub Releases](https://github.com/helloprtr/poly-prompt/releases)에서 플랫폼용 아카이브를 내려받아 `PATH`에 추가한 뒤, 위의 마지막 두 줄을 그대로 실행하면 됩니다.

실제 루프는 이렇게 시작합니다:

```bash
npm test 2>&1 | prtr go fix "왜 테스트가 깨지는지 정확한 원인만 찾아줘"
prtr swap gemini
prtr take patch --deep
prtr learn
```

`prtr demo`와 영어 `--dry-run` 흐름은 API 키 없이 바로 확인할 수 있습니다. 첫 실사용은 `prtr start`가 가장 편합니다.

## 핵심 루프

| 커맨드 | 역할 |
|---|---|
| `go` | 의도와 증거를 Claude, Codex, Gemini용 프롬프트로 정리 |
| `swap` | 마지막 실행을 다른 AI 앱으로 다시 전송 |
| `take` | 복사한 AI 답변을 다음 구조화된 액션으로 변환 |
| `take --deep` | 내부 멀티스텝 파이프라인을 거쳐 더 깊게 전달 |
| `again` | 마지막 실행을 같은 흐름으로 재실행 |
| `learn` | 번역 중에도 살아남는 리포 메모리와 보호 용어 구축 |
| `inspect` | 보내기 전에 라우팅 결과를 미리 확인 |

## 추가 시각 자료

| setup + doctor | delivery + paste |
|---|---|
| ![prtr setup과 doctor 데모](images/prtr-setup-doctor.gif) | ![prtr 전달과 붙여넣기 데모](images/prtr-delivery-paste.gif) |

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
