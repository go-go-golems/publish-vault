#!/usr/bin/env bash
# 02-http-slug-matrix.sh — probe a running retro-obsidian-publish for every
# spelling of the failing slug, so we can tell a slug mismatch apart from a
# routing/URL-decoding problem apart from "the note simply isn't loaded".
#
# Usage: ./02-http-slug-matrix.sh [base-url]
#   default base-url: http://127.0.0.1:18420
set -uo pipefail
BASE="${1:-http://127.0.0.1:18420}"
SLUG='transcripts/2026/08/09/designing-rag-abstractions/the_algebra_of_intervention_fields'
# The SSR sidecar builds its API URL with encodeURIComponent(slug), which
# percent-encodes the path separators. This row proves whether the Go router
# still matches once %2F is decoded.
ENC='transcripts%2F2026%2F08%2F09%2Fdesigning-rag-abstractions%2Fthe_algebra_of_intervention_fields'

probe() { # probe <label> <path>
  printf '%-34s %-6s %-10s %s\n' "$1" \
    "$(curl -s -o /dev/null -w '%{http_code}' --max-time 60 "$BASE$2")" \
    "$(curl -s -o /dev/null -w '%{size_download}' --max-time 60 "$BASE$2")" \
    "$2"
}

echo "base: $BASE"
printf '%-34s %-6s %-10s %s\n' LABEL HTTP BYTES PATH
probe "api raw slashes"        "/api/notes/$SLUG"
probe "api percent-encoded /"  "/api/notes/$ENC"
probe "api uppercase path"     "/api/notes/Transcripts/2026/08/09/Designing RAG Abstractions/The_Algebra_of_Intervention_Fields"
probe "api trailing slash"     "/api/notes/$SLUG/"
probe "api with .md suffix"    "/api/notes/$SLUG.md"
probe "api raw endpoint"       "/api/notes/$SLUG/raw"
probe "markdown mirror"        "/note/$SLUG.md"
probe "page route (SPA/SSR)"   "/note/$SLUG"
probe "api genuinely-missing"  "/api/notes/transcripts/2026/08/09/does-not-exist"
probe "config"                 "/api/config"
