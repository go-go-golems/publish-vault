#!/usr/bin/env python3
"""Group validator JSONL for local triage; the report can contain private paths."""
import argparse
import collections
import json
import os
import re
import sys


def category(row):
    rule, target = row["rule"], row["target"]
    if rule == "WIKI_HEADING_MISSING" and "#^" in target:
        return "block_reference"
    if rule == "WIKI_TARGET_MISSING" and (
        re.match(r"^\s*-[a-zA-Z]\s+", target)
        or re.match(r'''^\s*["']?\\?\$[A-Za-z_][A-Za-z_0-9]*''', target)
    ):
        return "possible_shell_test"
    return {
        "WIKI_TARGET_MISSING": "missing_target",
        "WIKI_HEADING_MISSING": "missing_heading",
        "WIKI_IMAGE_MISSING": "missing_image",
    }.get(rule, "other")


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("input", help="Validator JSONL input")
    ap.add_argument("output", help="Private JSON output path")
    args = ap.parse_args()
    groups = collections.defaultdict(list)
    with open(args.input, encoding="utf-8") as f:
        for line in f:
            if not line.strip():
                continue
            row = json.loads(line)
            groups[(category(row), row["rule"], row["target"])].append(
                {"path": row["path"], "line": row["line"], "column": row["column"]}
            )
    result = []
    for (kind, rule, target), occurrences in groups.items():
        occurrences.sort(key=lambda x: (x["path"], x["line"], x["column"]))
        result.append({"category": kind, "rule": rule, "target": target,
                       "count": len(occurrences), "occurrences": occurrences})
    result.sort(key=lambda x: (x["category"], -x["count"], x["target"]))
    report = {"note": "Heuristic categories require human review; missing does not prove a defect.",
              "total_occurrences": sum(x["count"] for x in result), "groups": result}
    # Restrictive mode on creation; reject an existing file to avoid overwriting it.
    fd = os.open(args.output, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, "w", encoding="utf-8") as f:
        json.dump(report, f, ensure_ascii=False, indent=2)
        f.write("\n")
    counts = collections.Counter()
    for g in result:
        counts[g["category"]] += g["count"]
    print(f"Grouped {report['total_occurrences']} occurrences into {len(result)} groups.")
    for kind, count in sorted(counts.items()):
        print(f"{kind}: {count}")
    print(f"Private report: {args.output}")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError) as exc:
        print(f"grouping failed: {exc}", file=sys.stderr)
        sys.exit(1)
