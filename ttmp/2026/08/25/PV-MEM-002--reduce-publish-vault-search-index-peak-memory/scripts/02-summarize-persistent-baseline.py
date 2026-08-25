#!/usr/bin/env python3
import argparse
import hashlib
import json
import pathlib
import statistics

parser = argparse.ArgumentParser()
parser.add_argument("baseline_dir")
parser.add_argument("output")
parser.add_argument("--publish-vault-commit", required=True)
parser.add_argument("--measure-commit", required=True)
parser.add_argument("--vault-commit", required=True)
parser.add_argument("--vault-ignore", required=True)
parser.add_argument("--publish-config")
parser.add_argument("--sample-interval-nanos", type=int, required=True)
args = parser.parse_args()
root = pathlib.Path(args.baseline_dir)


def sha256(path):
    return hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest()


def distribution(values):
    return {"min": min(values), "median": int(statistics.median(values)), "max": max(values)}


def read_total_bytes(trace_path):
    found = None
    with trace_path.open(encoding="utf-8") as stream:
        for line in stream:
            event = json.loads(line)
            if event.get("event_type") == "annotation" and event.get("phase") == "vault_walk_parse":
                attrs = {a["key"]: a["value"] for a in event.get("attributes", [])}
                found = int(attrs["total_bytes"])
    if found is None:
        raise ValueError(f"no vault_walk_parse total_bytes annotation in {trace_path}")
    return found


runs = []
for receipt_path in sorted(root.glob("run-*.receipt.json")):
    stem = receipt_path.name.removesuffix(".receipt.json")
    metadata = json.loads((root / f"{stem}.metadata.json").read_text(encoding="utf-8"))
    receipt = json.loads(receipt_path.read_text(encoding="utf-8"))
    phases = {phase["name"]: phase for phase in receipt["phases"]}
    runs.append(
        {
            "duration_nanos": receipt["duration_nanos"],
            "peaks": receipt["peaks"],
            "sources": receipt["sources"],
            "index_bytes": metadata["index_bytes"],
            "notes": metadata["notes"],
            "candidate_source_bytes": read_total_bytes(root / f"{stem}.jsonl"),
            "phases": phases,
        }
    )

if len(runs) != 3:
    raise SystemExit(f"want exactly 3 baseline receipts, got {len(runs)}")

phase_names = list(runs[0]["phases"])
for run in runs[1:]:
    if list(run["phases"]) != phase_names:
        raise SystemExit("phase order changed between baseline runs")

for field in ("notes", "candidate_source_bytes"):
    if len({run[field] for run in runs}) != 1:
        raise SystemExit(f"workload {field} changed between runs")

summary = {
    "schema_version": 1,
    "kind": "pv-mem-002-persistent-baseline",
    "instrumentation": {
        "publish_vault_commit": args.publish_vault_commit,
        "measure_commit": args.measure_commit,
        "sample_interval_nanos": args.sample_interval_nanos,
    },
    "workload": {
        "vault_commit": args.vault_commit,
        "vault_worktree_dirty": False,
        "vault_ignore_sha256": sha256(args.vault_ignore),
        "publish_config_present": args.publish_config is not None,
        "publish_config_sha256": sha256(args.publish_config) if args.publish_config else None,
        "published_notes": runs[0]["notes"],
        "candidate_source_bytes": runs[0]["candidate_source_bytes"],
        "markdown_candidates": runs[0]["phases"]["vault_walk_parse"]["progress"]["total"],
    },
    "persistent": {
        "runs": 3,
        "duration_nanos": distribution([run["duration_nanos"] for run in runs]),
        "peak_heap_alloc_bytes": distribution([run["peaks"]["heap_alloc_bytes"] for run in runs]),
        "peak_rss_bytes": distribution([run["peaks"]["rss_bytes"] for run in runs]),
        "peak_cgroup_current_bytes": distribution([run["peaks"]["cgroup_current_bytes"] for run in runs]),
        "index_bytes": distribution([run["index_bytes"] for run in runs]),
        "phases": {},
    },
    "privacy": {
        "contains_note_content": False,
        "allowed_trace_attributes": ["processed_notes", "processed_bytes", "total_bytes"],
    },
}

for name in phase_names:
    phases = [run["phases"][name] for run in runs]
    summary["persistent"]["phases"][name] = {
        "duration_nanos": distribution([phase["duration_nanos"] for phase in phases]),
        "peak_heap_alloc_bytes": distribution([phase["peaks"]["heap_alloc_bytes"] for phase in phases]),
        "peak_rss_bytes": distribution([phase["peaks"]["rss_bytes"] for phase in phases]),
        "peak_cgroup_current_bytes": distribution([phase["peaks"]["cgroup_current_bytes"] for phase in phases]),
        "processed": phases[0]["progress"]["processed"],
        "total": phases[0]["progress"]["total"],
    }

summary["dominant_phase_by_median_heap"] = max(
    phase_names,
    key=lambda name: summary["persistent"]["phases"][name]["peak_heap_alloc_bytes"]["median"],
)
summary["dominant_phase_by_median_rss"] = max(
    phase_names,
    key=lambda name: summary["persistent"]["phases"][name]["peak_rss_bytes"]["median"],
)

pathlib.Path(args.output).write_text(json.dumps(summary, indent=2, sort_keys=True) + "\n", encoding="utf-8")
