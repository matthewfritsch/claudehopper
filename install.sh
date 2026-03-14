#!/usr/bin/env bash
# Install ccswap globally via uv tool
set -euo pipefail

if command -v uv &>/dev/null; then
    echo "Installing ccswap via uv..."
    uv tool install claude-swap 2>/dev/null || uv tool install --from "git+https://github.com/$(whoami)/claude-swap.git" claude-swap
elif command -v pipx &>/dev/null; then
    echo "Installing ccswap via pipx..."
    pipx install claude-swap 2>/dev/null || pipx install "git+https://github.com/$(whoami)/claude-swap.git"
else
    echo "Error: uv or pipx required. Install uv: curl -LsSf https://astral.sh/uv/install.sh | sh"
    exit 1
fi

echo "Done! Run 'ccswap' to get started."
