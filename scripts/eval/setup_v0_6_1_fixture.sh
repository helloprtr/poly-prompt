#!/bin/zsh

set -euo pipefail

repo_root=$(cd "$(dirname "$0")/../.." && pwd)
fixture_root="$repo_root/examples/v0_6_1_eval_fixture"
target_root=/tmp/prtr-v0.6.1-eval

rm -rf "$target_root"
mkdir -p "$target_root"
cp -R "$fixture_root"/. "$target_root"

git init -q "$target_root"
git -C "$target_root" config user.email "fixture@example.com"
git -C "$target_root" config user.name "Fixture Bot"
git -C "$target_root" add .
git -C "$target_root" commit -qm "fixture base"

printf "\nPending checkout wording review.\n" >> "$target_root/README.md"
printf "Remember to compare Codex and Gemini outputs.\n" > "$target_root/notes.txt"

echo "fixture ready at $target_root"
