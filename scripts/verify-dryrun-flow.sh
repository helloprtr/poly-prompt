#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

PROMPT_TEXT="${PRTR_VERIFY_PROMPT:-왜 repo context가 비어 보이는지 root cause만 찾아줘}"
FAIL_LOG="${PRTR_VERIFY_FAIL_LOG:-FAIL: TestRepoContextCollect
repoctx: expected branch summary but got empty changes
--- FAIL: TestRepoContextCollect (0.00s)}"
GEMINI_REPLY="${PRTR_VERIFY_GEMINI_REPLY:-Root cause: repo context is not actually empty; it only looks sparse because the collector currently stores git status lines only. If you want richer context, update internal/repoctx/repoctx.go to include a short git diff summary and add regression coverage in internal/repoctx/repoctx_test.go. Keep the existing branch and change list, but add one more compact signal for why the repo matters to the prompt.}"
RUN_STAMP="$(date -u +"%Y%m%dT%H%M%SZ")"
DEFAULT_OUTPUT_DIR="${REPO_ROOT}/output/verify-dryrun/${RUN_STAMP}"
RUN_OUTPUT_DIR="${PRTR_VERIFY_OUTPUT_DIR:-${DEFAULT_OUTPUT_DIR}}"

TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/prtr-verify.XXXXXX")"
TMP_HOME="${TMP_ROOT}/home"
mkdir -p "${TMP_HOME}" "${TMP_HOME}/.config" "${RUN_OUTPUT_DIR}"

export HOME="${TMP_HOME}"
export XDG_CONFIG_HOME="${TMP_HOME}/.config"

CLIPBOARD_BACKEND=""
ORIGINAL_CLIPBOARD=""

detect_clipboard_backend() {
  if command -v pbcopy >/dev/null 2>&1 && command -v pbpaste >/dev/null 2>&1; then
    CLIPBOARD_BACKEND="pbcopy"
    return 0
  fi
  if command -v wl-copy >/dev/null 2>&1 && command -v wl-paste >/dev/null 2>&1; then
    CLIPBOARD_BACKEND="wl"
    return 0
  fi
  if command -v xclip >/dev/null 2>&1; then
    CLIPBOARD_BACKEND="xclip"
    return 0
  fi
  if command -v xsel >/dev/null 2>&1; then
    CLIPBOARD_BACKEND="xsel"
    return 0
  fi
  return 1
}

clipboard_read() {
  case "${CLIPBOARD_BACKEND}" in
    pbcopy) pbpaste ;;
    wl) wl-paste ;;
    xclip) xclip -selection clipboard -o ;;
    xsel) xsel --clipboard --output ;;
    *)
      return 1
      ;;
  esac
}

clipboard_write() {
  case "${CLIPBOARD_BACKEND}" in
    pbcopy) pbcopy ;;
    wl) wl-copy ;;
    xclip) xclip -selection clipboard ;;
    xsel) xsel --clipboard --input ;;
    *)
      return 1
      ;;
  esac
}

cleanup() {
  if [[ -n "${CLIPBOARD_BACKEND}" ]]; then
    printf '%s' "${ORIGINAL_CLIPBOARD}" | clipboard_write || true
  fi
}

trap cleanup EXIT

if ! detect_clipboard_backend; then
  echo "clipboard backend not found; install pbcopy/pbpaste, wl-clipboard, xclip, or xsel" >&2
  exit 1
fi

ORIGINAL_CLIPBOARD="$(clipboard_read 2>/dev/null || true)"

run_prtr() {
  (
    cd "${REPO_ROOT}"
    go run ./cmd/prtr "$@"
  )
}

write_stage_screen() {
  local title="$1"
  local command_text="$2"
  local stderr_file="$3"
  local stdout_file="$4"
  local output_file="$5"
  {
    printf '$ %s\n' "${command_text}"
    if [[ -s "${stderr_file}" ]]; then
      cat "${stderr_file}"
    fi
    if [[ -s "${stdout_file}" ]]; then
      printf '\n'
      cat "${stdout_file}"
    fi
  } > "${output_file}"
}

printf '%s' "${FAIL_LOG}" | run_prtr go fix --to claude --dry-run "${PROMPT_TEXT}" > "${TMP_ROOT}/go.stdout.txt" 2> "${TMP_ROOT}/go.stderr.txt"
run_prtr swap gemini --dry-run > "${TMP_ROOT}/swap.stdout.txt" 2> "${TMP_ROOT}/swap.stderr.txt"
printf '%s' "${GEMINI_REPLY}" | clipboard_write
run_prtr take patch --deep --dry-run > "${TMP_ROOT}/take.stdout.txt" 2> "${TMP_ROOT}/take.stderr.txt"
run_prtr history > "${TMP_ROOT}/history.txt"

write_stage_screen \
  "go -> claude" \
  "printf '%s' \"\$PRTR_VERIFY_FAIL_LOG\" | go run ./cmd/prtr go fix --to claude --dry-run \"\$PRTR_VERIFY_PROMPT\"" \
  "${TMP_ROOT}/go.stderr.txt" \
  "${TMP_ROOT}/go.stdout.txt" \
  "${TMP_ROOT}/go.screen.txt"

write_stage_screen \
  "swap -> gemini" \
  "go run ./cmd/prtr swap gemini --dry-run" \
  "${TMP_ROOT}/swap.stderr.txt" \
  "${TMP_ROOT}/swap.stdout.txt" \
  "${TMP_ROOT}/swap.screen.txt"

write_stage_screen \
  "take patch --deep" \
  "printf '%s' \"\$PRTR_VERIFY_GEMINI_REPLY\" | pbcopy && go run ./cmd/prtr take patch --deep --dry-run" \
  "${TMP_ROOT}/take.stderr.txt" \
  "${TMP_ROOT}/take.stdout.txt" \
  "${TMP_ROOT}/take.screen.txt"

{
  printf '$ go run ./cmd/prtr history\n'
  cat "${TMP_ROOT}/history.txt"
} > "${TMP_ROOT}/history.screen.txt"

HISTORY_PATH="${XDG_CONFIG_HOME}/prtr/history.json"

python3 - <<'PY' "${HISTORY_PATH}" "${TMP_ROOT}" "${RUN_OUTPUT_DIR}" > "${TMP_ROOT}/summary.txt"
import json
import pathlib
import sys
from datetime import datetime, timezone

history_path = pathlib.Path(sys.argv[1])
tmp_root = pathlib.Path(sys.argv[2])
run_output_dir = pathlib.Path(sys.argv[3])
entries = json.loads(history_path.read_text())

if len(entries) != 3:
    raise SystemExit(f"expected 3 history entries, got {len(entries)}")

latest = entries[-1]
artifact_root = pathlib.Path(latest["artifact_root"])
manifest = json.loads((artifact_root / "manifest.json").read_text())
lineage = json.loads((artifact_root / "lineage.json").read_text())
repo_context = json.loads((artifact_root / "evidence" / "repo_context.json").read_text())
patch_bundle = json.loads((artifact_root / "result" / "patch_bundle.json").read_text())
events_text = (artifact_root / "events.jsonl").read_text()
history_text = (tmp_root / "history.txt").read_text()

paths = {
    "go_stderr": tmp_root / "go.stderr.txt",
    "go_stdout": tmp_root / "go.stdout.txt",
    "go_screen": tmp_root / "go.screen.txt",
    "swap_stderr": tmp_root / "swap.stderr.txt",
    "swap_stdout": tmp_root / "swap.stdout.txt",
    "swap_screen": tmp_root / "swap.screen.txt",
    "take_stderr": tmp_root / "take.stderr.txt",
    "take_stdout": tmp_root / "take.stdout.txt",
    "take_screen": tmp_root / "take.screen.txt",
    "history_text": tmp_root / "history.txt",
    "history_screen": tmp_root / "history.screen.txt",
    "history_json": history_path,
    "manifest_json": artifact_root / "manifest.json",
    "lineage_json": artifact_root / "lineage.json",
    "repo_context_json": artifact_root / "evidence" / "repo_context.json",
    "patch_bundle_json": artifact_root / "result" / "patch_bundle.json",
    "events_jsonl": artifact_root / "events.jsonl",
}

copied = {}
for label, src in paths.items():
    suffix = src.suffix or ".txt"
    target = run_output_dir / f"{label}{suffix}"
    target.write_text(src.read_text())
    copied[label] = target

report_path = run_output_dir / "report.md"
summary_path = run_output_dir / "summary.txt"
summary_path.write_text("")

report_lines = [
    "# Dry-Run Flow Report",
    "",
    f"- Generated at (UTC): {datetime.now(timezone.utc).strftime('%Y-%m-%d %H:%M:%S UTC')}",
    f"- Repo: {repo_context.get('RepoName')}",
    f"- Branch: {repo_context.get('Branch')}",
    f"- History file: `{history_path}`",
    f"- Deep artifact root: `{artifact_root}`",
    "",
    "## Flow",
    "",
    "```text",
    "Claude dry-run -> Gemini swap dry-run -> Gemini patch deep dry-run",
    "```",
    "",
    "## 1. go -> claude",
    "",
    "Command:",
    "",
    "```bash",
    "printf '%s' \"$PRTR_VERIFY_FAIL_LOG\" | go run ./cmd/prtr go fix --to claude --dry-run \"$PRTR_VERIFY_PROMPT\"",
    "```",
    "",
    "stderr:",
    "",
    "```text",
    paths["go_stderr"].read_text().rstrip(),
    "```",
    "",
    "stdout:",
    "",
    "```text",
    paths["go_stdout"].read_text().rstrip(),
    "```",
    "",
    "## 2. swap -> gemini",
    "",
    "Command:",
    "",
    "```bash",
    "go run ./cmd/prtr swap gemini --dry-run",
    "```",
    "",
    "stderr:",
    "",
    "```text",
    paths["swap_stderr"].read_text().rstrip(),
    "```",
    "",
    "stdout:",
    "",
    "```text",
    paths["swap_stdout"].read_text().rstrip(),
    "```",
    "",
    "## 3. take patch --deep",
    "",
    "Command:",
    "",
    "```bash",
    "printf '%s' \"$PRTR_VERIFY_GEMINI_REPLY\" | pbcopy",
    "go run ./cmd/prtr take patch --deep --dry-run",
    "```",
    "",
    "stderr:",
    "",
    "```text",
    paths["take_stderr"].read_text().rstrip(),
    "```",
    "",
    "stdout:",
    "",
    "```text",
    paths["take_stdout"].read_text().rstrip(),
    "```",
    "",
    "## 4. History",
    "",
    "history output:",
    "",
    "```text",
    history_text.rstrip(),
    "```",
    "",
    "history.json excerpt:",
    "",
    "```json",
    json.dumps(entries, ensure_ascii=False, indent=2),
    "```",
    "",
    "## 5. Deep Artifacts",
    "",
    "manifest.json:",
    "",
    "```json",
    json.dumps(manifest, ensure_ascii=False, indent=2),
    "```",
    "",
    "lineage.json:",
    "",
    "```json",
    json.dumps(lineage, ensure_ascii=False, indent=2),
    "```",
    "",
    "repo_context.json:",
    "",
    "```json",
    json.dumps(repo_context, ensure_ascii=False, indent=2),
    "```",
    "",
    "patch_bundle.json:",
    "",
    "```json",
    json.dumps(patch_bundle, ensure_ascii=False, indent=2),
    "```",
    "",
    "events.jsonl:",
    "",
    "```json",
    events_text.rstrip(),
    "```",
    "",
    "## Saved Files",
    "",
]

for label, target in copied.items():
    report_lines.append(f"- `{label}`: `{target}`")

report_lines.append(f"- `report`: `{report_path}`")
report_lines.append("")

report_path.write_text("\n".join(report_lines) + "\n")

summary = []
summary.append("=== Paths ===")
summary.append(f"temp_home={tmp_root / 'home'}")
summary.append(f"history={history_path}")
summary.append(f"artifact_root={artifact_root}")
summary.append(f"output_dir={run_output_dir}")
summary.append("")

summary.append("=== History Entries ===")
for idx, entry in enumerate(entries, 1):
    summary.append(
        f"[{idx}] id={entry.get('id')} target={entry.get('target')} "
        f"parent={entry.get('parent_id', '')} engine={entry.get('engine', '')} "
        f"result={entry.get('result_type', '')} run_status={entry.get('run_status', '')}"
    )
summary.append("")
summary.append("=== Manifest ===")
summary.append(json.dumps(
    {
        "id": manifest.get("id"),
        "status": manifest.get("status"),
        "target_app": manifest.get("target_app"),
        "source_kind": manifest.get("source_kind"),
        "parent_history_id": manifest.get("parent_history_id"),
        "result_ref": manifest.get("result_ref"),
    },
    ensure_ascii=False,
    indent=2,
))
summary.append("")
summary.append("=== Lineage ===")
summary.append(json.dumps(lineage, ensure_ascii=False, indent=2))
summary.append("")
summary.append("=== Repo Context Evidence ===")
summary.append(json.dumps(repo_context, ensure_ascii=False, indent=2))
summary.append("")
summary.append("=== Patch Bundle ===")
summary.append(json.dumps(
    {
        "summary": patch_bundle.get("summary"),
        "risks": patch_bundle.get("risks"),
        "test_plan": patch_bundle.get("test_plan"),
    },
    ensure_ascii=False,
    indent=2,
))
summary.append("")
summary.append("=== Saved Files ===")
for label, target in copied.items():
    summary.append(f"{label}={target}")
summary.append(f"report={report_path}")

text = "\n".join(summary) + "\n"
summary_path.write_text(text)
print(text, end="")
PY

python3 "${SCRIPT_DIR}/render-terminal-shot.py" \
  --title "prtr verify · go -> claude" \
  --input "${RUN_OUTPUT_DIR}/go_screen.txt" \
  --output "${RUN_OUTPUT_DIR}/go_terminal.png"

python3 "${SCRIPT_DIR}/render-terminal-shot.py" \
  --title "prtr verify · swap -> gemini" \
  --input "${RUN_OUTPUT_DIR}/swap_screen.txt" \
  --output "${RUN_OUTPUT_DIR}/swap_terminal.png"

python3 "${SCRIPT_DIR}/render-terminal-shot.py" \
  --title "prtr verify · take patch --deep" \
  --input "${RUN_OUTPUT_DIR}/take_screen.txt" \
  --output "${RUN_OUTPUT_DIR}/take_terminal.png"

python3 "${SCRIPT_DIR}/render-terminal-shot.py" \
  --title "prtr verify · history" \
  --input "${RUN_OUTPUT_DIR}/history_screen.txt" \
  --output "${RUN_OUTPUT_DIR}/history_terminal.png"

{
  echo
  echo "## PNG Screenshots"
  echo
  echo "- \`go_terminal\`: \`${RUN_OUTPUT_DIR}/go_terminal.png\`"
  echo "- \`swap_terminal\`: \`${RUN_OUTPUT_DIR}/swap_terminal.png\`"
  echo "- \`take_terminal\`: \`${RUN_OUTPUT_DIR}/take_terminal.png\`"
  echo "- \`history_terminal\`: \`${RUN_OUTPUT_DIR}/history_terminal.png\`"
  echo
  echo "![go terminal](${RUN_OUTPUT_DIR}/go_terminal.png)"
  echo
  echo "![swap terminal](${RUN_OUTPUT_DIR}/swap_terminal.png)"
  echo
  echo "![take terminal](${RUN_OUTPUT_DIR}/take_terminal.png)"
  echo
  echo "![history terminal](${RUN_OUTPUT_DIR}/history_terminal.png)"
} >> "${RUN_OUTPUT_DIR}/report.md"

{
  echo
  echo "=== PNG Screenshots ==="
  echo "go_terminal=${RUN_OUTPUT_DIR}/go_terminal.png"
  echo "swap_terminal=${RUN_OUTPUT_DIR}/swap_terminal.png"
  echo "take_terminal=${RUN_OUTPUT_DIR}/take_terminal.png"
  echo "history_terminal=${RUN_OUTPUT_DIR}/history_terminal.png"
} >> "${TMP_ROOT}/summary.txt"

echo "=== go output ==="
cat "${TMP_ROOT}/go.stdout.txt"
echo
echo "=== swap output ==="
cat "${TMP_ROOT}/swap.stdout.txt"
echo
echo "=== take output ==="
cat "${TMP_ROOT}/take.stdout.txt"
echo
cat "${TMP_ROOT}/summary.txt"
echo
echo "=== events.jsonl ==="
cat "$(python3 - <<'PY' "${HISTORY_PATH}"
import json
import pathlib
import sys

entries = json.loads(pathlib.Path(sys.argv[1]).read_text())
print(pathlib.Path(entries[-1]["artifact_root"]) / "events.jsonl")
PY
)"
echo
echo "=== PNG Screenshots ==="
echo "go_terminal=${RUN_OUTPUT_DIR}/go_terminal.png"
echo "swap_terminal=${RUN_OUTPUT_DIR}/swap_terminal.png"
echo "take_terminal=${RUN_OUTPUT_DIR}/take_terminal.png"
echo "history_terminal=${RUN_OUTPUT_DIR}/history_terminal.png"
