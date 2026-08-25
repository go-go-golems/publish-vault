#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 PUBLISH_BIN MEASURE_BIN OUTPUT_DIR" >&2
  exit 2
fi
publish_bin=$(realpath "$1")
measure_bin=$(realpath "$2")
output_dir=$(realpath -m "$3")
script_dir=$(cd "$(dirname "$0")" && pwd)
mkdir -p "$output_dir"

cases=(
  "100:1024"
  "160:8192"
  "500:8192"
  "1000:8192"
  "200:32768"
)
case_number=0
for spec in "${cases[@]}"; do
  case_number=$((case_number + 1))
  documents=${spec%%:*}
  payload_bytes=${spec##*:}
  name="docs-${documents}-bytes-${payload_bytes}"
  work=$(mktemp -d "${TMPDIR:-/tmp}/pv-mem-002-scaling.XXXXXX")
  trap 'rm -rf "$work"' EXIT INT TERM
  python3 - "$work/vault" "$documents" "$payload_bytes" <<'PY'
import pathlib,sys
root=pathlib.Path(sys.argv[1]); documents=int(sys.argv[2]); payload_bytes=int(sys.argv[3]); root.mkdir(parents=True)
unit="bounded-memory-payload "
payload=(unit*(payload_bytes//len(unit)+1))[:payload_bytes]
for i in range(documents):
    (root/f"note-{i:05d}.md").write_text(f"# Generated {i:05d}\n\nfixture-token-{i:05d} {payload}\n",encoding="utf-8")
PY
  BASE_PORT=$((28500 + case_number * 10)) MEASURE_INTERVAL=100ms \
    "$script_dir/01-run-persistent-baseline.sh" "$publish_bin" "$measure_bin" "$work/vault" "$output_dir/$name" 1 >/dev/null
  python3 - "$output_dir/$name/run-1.metadata.json" "$documents" "$payload_bytes" >"$output_dir/$name/result.json" <<'PY'
import json,sys
m=json.load(open(sys.argv[1])); phase=next(p for p in m["phases"] if p["name"]=="search_index")
print(json.dumps({"schema_version":1,"documents":int(sys.argv[2]),"payload_bytes_per_document":int(sys.argv[3]),"source_payload_bytes":int(sys.argv[2])*int(sys.argv[3]),"duration_nanos":m["duration_nanos"],"index_bytes":m["index_bytes"],"peak_heap_alloc_bytes":m["peaks"]["heap_alloc_bytes"],"peak_rss_bytes":m["peaks"]["rss_bytes"],"search_index":{"duration_nanos":phase["duration_nanos"],"peak_heap_alloc_bytes":phase["peaks"]["heap_alloc_bytes"],"peak_rss_bytes":phase["peaks"]["rss_bytes"]}},indent=2,sort_keys=True))
PY
  rm -rf "$work"
  trap - EXIT INT TERM
done

python3 - "$output_dir" <<'PY'
import json,pathlib,sys
root=pathlib.Path(sys.argv[1]); rows=[]
for path in sorted(root.glob("*/result.json")):
    rows.append(json.loads(path.read_text()))
(root/"summary.json").write_text(json.dumps({"schema_version":1,"cases":rows},indent=2,sort_keys=True)+"\n")
PY
