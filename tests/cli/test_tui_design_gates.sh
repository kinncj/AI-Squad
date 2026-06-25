#!/usr/bin/env bash
# Proves the SDLC design gates pass on terminal (tui) artifacts unchanged.
set -uo pipefail
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
ROOT="${REPO_ROOT:-$PWD}"
cd "$TMP" || exit 1

mkdir -p docs/stories docs/design/wireframes docs/design/mockups
cat > docs/stories/STORY-TUI.md <<'EOF'
---
id: "STORY-TUI"
title: "Gate status overlay"
ui: true
phase: validate
---
Scenario: shows gate status
EOF
printf 'status: approved\n# wireframe\n' > docs/design/wireframes/STORY-TUI.wireframe.md
printf 'status: approved\n# mockup\n'    > docs/design/mockups/STORY-TUI.mockup.md
cat > docs/design/mockups/STORY-TUI.a11y.json <<'EOF'
{ "target": "tui", "violations": [ { "id": "no-color-support", "impact": "moderate" } ], "passes": [] }
EOF

bash "$ROOT/scripts/sdlc/design-approved-gate.sh" docs/stories/STORY-TUI.md || { echo "FAIL design gate"; exit 1; }
bash "$ROOT/scripts/sdlc/a11y-gate.sh"            docs/stories/STORY-TUI.md || { echo "FAIL a11y gate"; exit 1; }

# Negative: a critical violation must FAIL the a11y gate.
cat > docs/design/mockups/STORY-TUI.a11y.json <<'EOF'
{ "target": "tui", "violations": [ { "id": "keyboard-reachable", "impact": "critical" } ] }
EOF
if bash "$ROOT/scripts/sdlc/a11y-gate.sh" docs/stories/STORY-TUI.md; then echo "FAIL: critical should block"; exit 1; fi

echo "PASS tui design gates"
