# prtr watch hook — managed by prtr, do not edit manually
# Installed by: prtr watch

_prtr_preexec() {
  _PRTR_CMD="$1"
}

_prtr_precmd() {
  local exit_code=$?
  local suggest_file="$HOME/.config/prtr/watch-suggest"

  if [[ -f "$suggest_file" ]]; then
    local action
    action=$(python3 -c "import json; d=json.load(open('$suggest_file')); print(d.get('action','fix'))" 2>/dev/null || echo "fix")
    echo ""
    echo "⚡ prtr: context ready"
    python3 -c "
import json
d=json.load(open('$suggest_file'))
for line in d.get('context_lines', []):
    print('  •', line)
" 2>/dev/null
    printf "  → prtr go %s [y/N] " "$action"
    read -r prtr_reply
    rm -f "$suggest_file"
    if [[ "$prtr_reply" == "y" || "$prtr_reply" == "Y" ]]; then
      prtr go "$action"
    fi
  fi

  if [[ -n "$_PRTR_CMD" ]]; then
    local sock="$HOME/.config/prtr/watch.sock"
    local payload="{\"exit_code\":$exit_code,\"cmd\":\"$_PRTR_CMD\",\"output_file\":\"$TMPDIR/prtr-last-output\"}"

    if command -v socat &>/dev/null && [[ -S "$sock" ]]; then
      printf '%s\n' "$payload" | socat - UNIX-CONNECT:"$sock" 2>/dev/null || true
    else
      printf '%s\n' "$payload" > "$TMPDIR/prtr-watch-event" 2>/dev/null || true
    fi
    unset _PRTR_CMD
  fi
}

autoload -Uz add-zsh-hook
add-zsh-hook preexec _prtr_preexec
add-zsh-hook precmd _prtr_precmd
