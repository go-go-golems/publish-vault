#!/usr/bin/env python3
import argparse
import json
import pathlib

parser = argparse.ArgumentParser()
parser.add_argument("matrix_dir")
parser.add_argument("output")
args = parser.parse_args()
root = pathlib.Path(args.matrix_dir)

rows = []
for directory in sorted(path for path in root.iterdir() if path.is_dir()):
    metadata = json.loads((directory / "metadata.json").read_text(encoding="utf-8"))
    summary = metadata["summary"]
    phase = summary["Phases"][0]
    rows.append(
        {
            "name": directory.name,
            "batch_documents": metadata["batch_documents"],
            "batch_bytes": metadata["batch_bytes"],
            "documents": metadata["documents"],
            "indexed_bytes": metadata["indexed_bytes"],
            "duration_nanos": metadata["duration_nanos"],
            "peak_heap_alloc_bytes": phase["Peaks"]["heap_alloc_bytes"],
            "peak_rss_bytes": phase["Peaks"]["rss_bytes"],
            "total_alloc_bytes": summary["CounterDelta"]["total_alloc_bytes"],
            "gc_cycles": summary["CounterDelta"]["num_gc"],
            "index_bytes": metadata["index_bytes"],
        }
    )

by_name = {row["name"]: row for row in rows}
baseline = by_name["current"]
for row in rows:
    row["versus_current_percent"] = {
        field: round((row[field] / baseline[field] - 1) * 100, 2)
        for field in ("duration_nanos", "peak_heap_alloc_bytes", "peak_rss_bytes", "total_alloc_bytes", "index_bytes")
    }

selected = by_name["batch16"]
result = {
    "schema_version": 1,
    "runs_are_single_exploratory_measurements": True,
    "rows": rows,
    "selected": {
        "name": selected["name"],
        "batch_documents": selected["batch_documents"],
        "batch_bytes": selected["batch_bytes"],
        "rationale": "best observed heap/RSS combination while also reducing duration, cumulative allocation, GC cycles, and index bytes",
    },
}
pathlib.Path(args.output).write_text(json.dumps(result, indent=2, sort_keys=True) + "\n", encoding="utf-8")
