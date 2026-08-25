#!/usr/bin/env python3
import argparse
import json
import pathlib

parser = argparse.ArgumentParser()
parser.add_argument("baseline")
parser.add_argument("candidate")
parser.add_argument("output")
args = parser.parse_args()
baseline = json.loads(pathlib.Path(args.baseline).read_text(encoding="utf-8"))
candidate = json.loads(pathlib.Path(args.candidate).read_text(encoding="utf-8"))

workload_fields = (
    "vault_commit",
    "vault_ignore_sha256",
    "publish_config_present",
    "publish_config_sha256",
    "published_notes",
    "markdown_candidates",
    "candidate_source_bytes",
)
for field in workload_fields:
    if baseline["workload"][field] != candidate["workload"][field]:
        raise SystemExit(f"workload mismatch for {field}: {baseline['workload'][field]!r} != {candidate['workload'][field]!r}")


def comparison(before, after):
    return {
        "baseline": before,
        "candidate": after,
        "absolute_change": after - before,
        "percent_change": round((after / before - 1) * 100, 2),
    }


b = baseline["persistent"]
c = candidate["persistent"]
metrics = {}
for field in ("duration_nanos", "peak_heap_alloc_bytes", "peak_rss_bytes", "index_bytes"):
    metrics[field] = comparison(b[field]["median"], c[field]["median"])

phase_metrics = {}
for phase in b["phases"]:
    phase_metrics[phase] = {}
    for field in ("duration_nanos", "peak_heap_alloc_bytes", "peak_rss_bytes"):
        phase_metrics[phase][field] = comparison(b["phases"][phase][field]["median"], c["phases"][phase][field]["median"])

notes = baseline["workload"]["published_notes"]
metrics["throughput_notes_per_second"] = comparison(
    notes / (b["duration_nanos"]["median"] / 1e9),
    notes / (c["duration_nanos"]["median"] / 1e9),
)
result = {
    "schema_version": 1,
    "workload": {field: baseline["workload"][field] for field in workload_fields},
    "baseline_commit": baseline["instrumentation"]["publish_vault_commit"],
    "candidate_commit": candidate["instrumentation"]["publish_vault_commit"],
    "runs_each": 3,
    "metrics": metrics,
    "phases": phase_metrics,
    "baseline_dominant_phase_by_heap": baseline["dominant_phase_by_median_heap"],
    "candidate_dominant_phase_by_heap": candidate["dominant_phase_by_median_heap"],
    "baseline_dominant_phase_by_rss": baseline["dominant_phase_by_median_rss"],
    "candidate_dominant_phase_by_rss": candidate["dominant_phase_by_median_rss"],
}
pathlib.Path(args.output).write_text(json.dumps(result, indent=2, sort_keys=True) + "\n", encoding="utf-8")
