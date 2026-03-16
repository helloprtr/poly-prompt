#!/bin/zsh

set -euo pipefail

repo_root=$(cd "$(dirname "$0")/../.." && pwd)
demo_root=/tmp/prtr-demo
workspace="$demo_root/workspace"
bin_dir="$demo_root/bin"
config_dir="$demo_root/config"
user_config_dir="$config_dir/prtr"
data_dir="$demo_root/data"

rm -rf "$demo_root"
mkdir -p "$workspace/docs" "$bin_dir" "$user_config_dir" "$data_dir"

cd "$repo_root"
go build -o "$bin_dir/prtr" ./cmd/prtr

cat > "$workspace/README.md" <<'EOF'
# Demo Checkout

prtr turns what you mean into the next AI action.

- Start fast with `go`
- Compare apps with `swap`
- Turn answers into work with `take`
- Keep repo memory local with `learn`
EOF

cat > "$workspace/docs/release.md" <<'EOF'
# Release Notes Draft

- Keep the loop visible.
- Show the same intent across Claude, Codex, and Gemini.
- Preserve repo vocabulary during follow-up runs.
EOF

git init -q "$workspace"
git -C "$workspace" config user.email "demo@example.com"
git -C "$workspace" config user.name "Demo Bot"
git -C "$workspace" add README.md docs/release.md
git -C "$workspace" commit -qm "demo base"

printf "\nPending release notes refresh.\n" >> "$workspace/README.md"
printf "Track follow-up demo edits.\n" > "$workspace/notes.txt"

cat > "$user_config_dir/config.toml" <<'EOF'
deepl_api_key = ""
translation_source_lang = "auto"
translation_target_lang = "en"
default_target = "claude"
default_template_preset = "claude-structured"

[routing]
enabled = true
policy = "deterministic-v1"

[routing.fixed_targets]
ask = ""
review = ""
fix = ""
design = ""

[routing.mode_defaults]
ask = "claude"
review = "claude"
fix = "codex"
design = "gemini"
EOF

echo "demo environment ready at $demo_root"
