#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 5 ]]; then
  echo "usage: $0 PUBLISH_BIN MEASURE_BIN VAULT_DIR OUTPUT_DIR RUNS" >&2
  exit 2
fi

publish_bin=$(realpath "$1")
measure_bin=$(realpath "$2")
vault_dir=$(realpath "$3")
output_dir=$(realpath -m "$4")
runs=$5
interval=${MEASURE_INTERVAL:-100ms}
base_port=${BASE_PORT:-28180}

if ! [[ "$runs" =~ ^[1-9][0-9]*$ ]]; then
  echo "RUNS must be a positive integer" >&2
  exit 2
fi
mkdir -p "$output_dir"

for run in $(seq 1 "$runs"); do
  work=$(mktemp -d "${TMPDIR:-/tmp}/pv-mem-002-baseline.XXXXXX")
  pid=""
  cleanup() {
    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
      kill -TERM "$pid" 2>/dev/null || true
      wait "$pid" 2>/dev/null || true
    fi
    rm -rf "$work"
  }
  trap cleanup EXIT INT TERM

  trace_dir="$work/traces"
  search_dir="$work/search"
  port=$((base_port + run))
  "$publish_bin" serve \
    --vault "$vault_dir" \
    --serve-web=false \
    --watch=false \
    --port "$port" \
    --measure-interval "$interval" \
    --measure-trace-dir "$trace_dir" \
    --search-index-path "$search_dir" \
    >"$work/server.log" 2>&1 &
  pid=$!

  ready=false
  for _ in $(seq 1 1800); do
    if curl -fsS "http://127.0.0.1:$port/api/healthz" >"$work/health.json" 2>/dev/null; then
      ready=true
      break
    fi
    if ! kill -0 "$pid" 2>/dev/null; then
      break
    fi
    sleep 0.1
  done
  if [[ "$ready" != true ]]; then
    echo "run $run did not become ready" >&2
    tail -100 "$work/server.log" >&2 || true
    exit 1
  fi

  index_bytes=$(du -sb "$search_dir" | awk '{print $1}')
  kill -TERM "$pid" 2>/dev/null || true
  wait "$pid" 2>/dev/null || true
  pid=""

  mapfile -t traces < <(find "$trace_dir" -maxdepth 1 -name '*.jsonl' -type f -print)
  mapfile -t receipts < <(find "$trace_dir" -maxdepth 1 -name '*.receipt.json' -type f -print)
  if [[ ${#traces[@]} -ne 1 || ${#receipts[@]} -ne 1 ]]; then
    echo "run $run expected one trace and receipt; got ${#traces[@]} trace(s), ${#receipts[@]} receipt(s)" >&2
    exit 1
  fi

  cp "${traces[0]}" "$output_dir/run-${run}.jsonl"
  cp "${receipts[0]}" "$output_dir/run-${run}.receipt.json"
  "$measure_bin" summarize "$output_dir/run-${run}.jsonl" --format jsonl >"$output_dir/run-${run}.summary.jsonl"
  python3 - "$work/health.json" "$output_dir/run-${run}.receipt.json" "$output_dir/run-${run}.metadata.json" "$run" "$index_bytes" <<'PY'
import json
import pathlib
import sys

health = json.load(open(sys.argv[1], encoding="utf-8"))
receipt = json.load(open(sys.argv[2], encoding="utf-8"))
metadata = {
    "run": int(sys.argv[4]),
    "mode": "persistent",
    "notes": health["notes"],
    "duration_nanos": receipt["duration_nanos"],
    "index_bytes": int(sys.argv[5]),
    "peaks": receipt["peaks"],
    "sources": receipt["sources"],
    "phases": [
        {
            "name": phase["name"],
            "duration_nanos": phase["duration_nanos"],
            "processed": phase["progress"]["processed"],
            "total": phase["progress"]["total"],
            "peaks": phase["peaks"],
        }
        for phase in receipt.get("phases", [])
    ],
}
pathlib.Path(sys.argv[3]).write_text(json.dumps(metadata, indent=2, sort_keys=True) + "\n", encoding="utf-8")
print(json.dumps(metadata, sort_keys=True))
PY

  trap - EXIT INT TERM
  rm -rf "$work"
done
