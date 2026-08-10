#!/usr/bin/env bash
# 01-measure-vault-on-disk.sh
#
# Measures the ON-DISK footprint of an Obsidian vault so the in-process RSS
# numbers from 02-measure-vault-memory can be expressed as a multiplier of the
# raw markdown bytes.
#
# Usage:
#   ./01-measure-vault-on-disk.sh [VAULT_PATH]
#
# Default VAULT_PATH is Manuel's go-go-parc vault.
set -euo pipefail

VAULT="${1:-/home/manuel/code/wesen/go-go-golems/go-go-parc}"

if [ ! -d "$VAULT" ]; then
  echo "ERROR: vault directory does not exist: $VAULT" >&2
  exit 1
fi

echo "vault: $VAULT"
echo

echo "== total size (including .git) =="
du -sh "$VAULT"

echo
echo "== total size (excluding .git) =="
du -sh --exclude=.git "$VAULT"

echo
echo "== markdown file count (excluding .git) =="
find "$VAULT" -name '.git' -prune -o -name '*.md' -print | wc -l

echo
echo "== markdown total bytes =="
find "$VAULT" -name '.git' -prune -o -name '*.md' -print0 \
  | du -cb --files0-from=- 2>/dev/null | tail -1

echo
echo "== non-markdown (asset) file count =="
find "$VAULT" -name '.git' -prune -o -type f ! -name '*.md' -print | wc -l

echo
echo "== non-markdown (asset) total bytes =="
find "$VAULT" -name '.git' -prune -o -type f ! -name '*.md' -print0 \
  | du -cb --files0-from=- 2>/dev/null | tail -1

echo
echo "== 20 largest markdown notes =="
find "$VAULT" -name '.git' -prune -o -name '*.md' -printf '%s %p\n' \
  | sort -rn | head -20

echo
echo "== markdown size histogram (bytes) =="
find "$VAULT" -name '.git' -prune -o -name '*.md' -printf '%s\n' \
  | sort -n | awk '
    { a[NR] = $1; sum += $1 }
    END {
      printf "count=%d\n", NR
      printf "sum=%d\n", sum
      printf "mean=%.0f\n", sum / NR
      printf "p50=%d\n", a[int(NR*0.50)]
      printf "p90=%d\n", a[int(NR*0.90)]
      printf "p99=%d\n", a[int(NR*0.99)]
      printf "max=%d\n", a[NR]
    }'
