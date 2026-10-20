#!/usr/bin/env bash
# Installs the Claude agent team config into this repo.
# Run from the repo root:  bash agent-setup/install.sh   (add --dry-run to preview)
set -euo pipefail
DRY=${1:-}
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/.." && pwd)"
run() { if [[ "$DRY" == "--dry-run" ]]; then echo "[dry-run] $*"; else "$@"; fi; }
for f in "$root/.claude/settings.json" "$root/.mcp.json"; do
  [[ -e "$f" ]] && echo "NOTE: $f exists and will be overwritten (backup: $f.bak)" && run cp "$f" "$f.bak"
done
run mkdir -p "$root/.claude" "$root/.github"
run cp -R "$here/dot-claude/." "$root/.claude/"
run cp "$here/dot-github/pull_request_template.md" "$root/.github/"
run cp "$here/dot-mcp.json" "$root/.mcp.json"
echo "Installed. Next: review the files, then 'git add .claude .github .mcp.json' and you can delete agent-setup/."
