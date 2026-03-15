#!/usr/bin/env bash
# sandbox.sh — drop into a shell with claudehopper pointing at temp directories.
# Your real ~/.claude and ~/.config/claudehopper are never touched.
#
# Usage:
#   ./scripts/sandbox.sh          # interactive shell
#   ./scripts/sandbox.sh hop list # run a single command
#
# The sandbox creates:
#   /tmp/hop-sandbox-XXXX/claude/     → fake ~/.claude
#   /tmp/hop-sandbox-XXXX/hopper/     → fake ~/.config/claudehopper
#
# Both are cleaned up on exit unless KEEP_SANDBOX=1 is set.

set -euo pipefail

SANDBOX=$(mktemp -d /tmp/hop-sandbox-XXXX)
export CLAUDE_DIR="$SANDBOX/claude"
export CLAUDEHOPPER_HOME="$SANDBOX/hopper"

mkdir -p "$CLAUDE_DIR" "$CLAUDEHOPPER_HOME"

# Seed a minimal fake ~/.claude so from-current has something to capture
cat > "$CLAUDE_DIR/settings.json" <<'SETTINGS'
{"editor": "sandbox"}
SETTINGS
cat > "$CLAUDE_DIR/CLAUDE.md" <<'MD'
# Sandbox
This is a test CLAUDE.md created by the sandbox script.
MD

echo "╔══════════════════════════════════════════════╗"
echo "║  claudehopper sandbox                       ║"
echo "╚══════════════════════════════════════════════╝"
echo ""
echo "  CLAUDE_DIR=$CLAUDE_DIR"
echo "  CLAUDEHOPPER_HOME=$CLAUDEHOPPER_HOME"
echo ""
echo "  Your real ~/.claude is untouched."
echo "  Type 'exit' to leave. Sandbox is cleaned up automatically."
echo ""

# Build fresh binary
go build -o "$SANDBOX/hop" . 2>/dev/null && export PATH="$SANDBOX:$PATH"

cleanup() {
    if [[ "${KEEP_SANDBOX:-}" == "1" ]]; then
        echo "Sandbox preserved at: $SANDBOX"
    else
        rm -rf "$SANDBOX"
    fi
}
trap cleanup EXIT

if [[ $# -gt 0 ]]; then
    # Run a single command
    "$@"
else
    # Interactive shell
    PS1="(hop-sandbox) \$ " bash --norc --noprofile
fi
