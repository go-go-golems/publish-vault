#!/usr/bin/env python3
import argparse
import json
import pathlib
import re

parser = argparse.ArgumentParser()
parser.add_argument("artifact_dir")
parser.add_argument("output")
args = parser.parse_args()
root = pathlib.Path(args.artifact_dir)


def parse_profile(path):
    text = path.read_text(encoding="utf-8")
    profile_type = re.search(r"^Type: (\S+)$", text, re.MULTILINE).group(1)
    total = int(re.search(r" of ([0-9]+)B total$", text, re.MULTILINE).group(1))
    rows = []
    row_re = re.compile(r"^\s*([0-9]+)B\s+[0-9.]+%\s+[0-9.]+%\s+([0-9]+)B\s+[0-9.]+%\s+(.+)$")
    for line in text.splitlines():
        match = row_re.match(line)
        if match:
            rows.append({"flat_bytes": int(match.group(1)), "cumulative_bytes": int(match.group(2)), "function": match.group(3)})
    return {"type": profile_type, "total_bytes": total, "top": rows[:15]}


checkpoints = []
for path in sorted(root.glob("checkpoint-*.json")):
    value = json.loads(path.read_text(encoding="utf-8"))
    memory = value["memory"]
    checkpoints.append(
        {
            "percent": value["percent"],
            "processed_documents": value["processed_documents"],
            "indexed_bytes": value["indexed_bytes"],
            "elapsed_nanos": value["elapsed_nanos"],
            "forced_gc": value["forced_gc"],
            "heap_alloc_bytes": memory["heap_alloc_bytes"],
            "heap_sys_bytes": memory["heap_sys_bytes"],
            "runtime_sys_bytes": memory["runtime_sys_bytes"],
            "rss_bytes": memory["rss_bytes"],
            "rss_peak_bytes": memory["rss_peak_bytes"],
            "pss_bytes": memory["pss_bytes"],
            "process_anonymous_bytes": memory["process_anonymous_bytes"],
            "process_private_dirty_bytes": memory["process_private_dirty_bytes"],
            "process_private_clean_bytes": memory["process_private_clean_bytes"],
            "num_gc": memory["num_gc"],
            "gc_goal_bytes": memory["gc_goal_bytes"],
            "total_alloc_bytes": memory["total_alloc_bytes"],
            "cgroup_availability": memory["cgroup_availability"],
        }
    )

profiles = {}
for percent in (0, 25, 50, 75, 100):
    key = f"{percent:03d}"
    profiles[key] = {
        "inuse_space": parse_profile(root / f"pprof-{key}-inuse-space.txt"),
        "alloc_space": parse_profile(root / f"pprof-{key}-alloc-space.txt"),
    }

result = {
    "schema_version": 1,
    "diagnostic_perturbed_by_forced_gc": True,
    "raw_profiles_retained": False,
    "checkpoints": checkpoints,
    "profiles": profiles,
    "findings": {
        "retained_heap_growth_0_to_100_bytes": checkpoints[-1]["heap_alloc_bytes"] - checkpoints[0]["heap_alloc_bytes"],
        "heap_sys_growth_0_to_100_bytes": checkpoints[-1]["heap_sys_bytes"] - checkpoints[0]["heap_sys_bytes"],
        "rss_growth_0_to_100_bytes": checkpoints[-1]["rss_bytes"] - checkpoints[0]["rss_bytes"],
        "total_allocation_growth_0_to_100_bytes": checkpoints[-1]["total_alloc_bytes"] - checkpoints[0]["total_alloc_bytes"],
        "selected_hypothesis": "bounded Bleve batches can reduce per-document segment and merge allocation churn",
    },
}
pathlib.Path(args.output).write_text(json.dumps(result, indent=2, sort_keys=True) + "\n", encoding="utf-8")
