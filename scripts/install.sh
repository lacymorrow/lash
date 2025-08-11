#!/usr/bin/env bash
set -euo pipefail

prefix=${PREFIX:-/usr/local}
bindir="$prefix/bin"
zshdir="$prefix/share/zsh/site-functions"
plugdir="$prefix/share/lash"

echo "Building lash-agent..."
mkdir -p "$bindir"
go build -o "$bindir/lash-agent" ./cmd/lash-agent

echo "Building crush-onboard..."
go build -o "$bindir/crush-onboard" ./cmd/crush-onboard

echo "Installing zsh plugin..."
mkdir -p "$plugdir"
cp -f scripts/zsh/lash.plugin.zsh "$plugdir/lash.plugin.zsh"

echo "Done. To enable in zsh, add to ~/.zshrc:"
echo "  source $plugdir/lash.plugin.zsh"


