#!/usr/bin/env python3
"""Audit Go Go Golems repositories for org-style tooling drift.

This script is intentionally read-only. It scans first-level repositories under
~/code/wesen/go-go-golems by default and emits a Markdown table that helps plan
safe modernization batches.
"""

from __future__ import annotations

import argparse
import json
import subprocess
from dataclasses import asdict, dataclass
from pathlib import Path

TARGET_GO = "1.26.4"
TARGET_GOLANGCI = "v2.12.2"


@dataclass
class RepoAudit:
    name: str
    path: str
    dirty: bool
    go_version: str
    golangci_version: str
    has_golangci_config: bool
    has_makefile: bool
    has_glazed: bool
    has_logcopter: bool
    has_github_workflows: bool
    classification: str
    notes: str


def read_text(path: Path) -> str:
    try:
        return path.read_text()
    except FileNotFoundError:
        return ""


def git_dirty(repo: Path) -> bool:
    try:
        out = subprocess.check_output(
            ["git", "status", "--short"], cwd=repo, text=True, stderr=subprocess.DEVNULL
        )
    except Exception:
        return True
    return bool(out.strip())


def parse_go_version(go_mod: str) -> str:
    for line in go_mod.splitlines():
        if line.startswith("go "):
            return line.split()[1]
    return ""


def classify(audit: RepoAudit) -> RepoAudit:
    notes: list[str] = []
    if audit.dirty:
        notes.append("dirty working tree")
    if not audit.go_version:
        notes.append("missing go directive")
    elif audit.go_version != TARGET_GO:
        notes.append(f"go {audit.go_version} != {TARGET_GO}")
    if audit.golangci_version and audit.golangci_version != TARGET_GOLANGCI:
        notes.append(f"golangci {audit.golangci_version} != {TARGET_GOLANGCI}")
    if not audit.golangci_version:
        notes.append("missing .golangci-lint-version")
    if not audit.has_golangci_config:
        notes.append("missing .golangci.yml")
    if not audit.has_makefile:
        notes.append("missing Makefile")

    if audit.dirty:
        classification = "manual-review-dirty"
    elif audit.go_version == TARGET_GO and audit.golangci_version == TARGET_GOLANGCI:
        classification = "current-or-nearly-current"
    elif audit.go_version and audit.go_version < "1.25":
        classification = "legacy-manual-review"
    else:
        classification = "safe-bump-candidate"

    audit.classification = classification
    audit.notes = "; ".join(notes) if notes else "ok"
    return audit


def audit_repo(repo: Path) -> RepoAudit | None:
    go_mod_path = repo / "go.mod"
    if not go_mod_path.exists():
        return None
    go_mod = read_text(go_mod_path)
    golangci_version = read_text(repo / ".golangci-lint-version").strip()
    audit = RepoAudit(
        name=repo.name,
        path=str(repo),
        dirty=git_dirty(repo),
        go_version=parse_go_version(go_mod),
        golangci_version=golangci_version,
        has_golangci_config=(repo / ".golangci.yml").exists(),
        has_makefile=(repo / "Makefile").exists(),
        has_glazed="github.com/go-go-golems/glazed" in go_mod,
        has_logcopter="github.com/go-go-golems/logcopter" in go_mod,
        has_github_workflows=(repo / ".github" / "workflows").exists(),
        classification="",
        notes="",
    )
    return classify(audit)


def markdown_table(audits: list[RepoAudit]) -> str:
    lines = [
        "---",
        "title: Org Style Audit Report",
        "doc_type: reference",
        "status: active",
        "intent: short-term",
        "topics:",
        "  - tooling",
        "  - ci",
        "  - linting",
        "---",
        "",
        "# Org Style Audit Report",
        "",
        f"Target Go: `{TARGET_GO}`",
        f"Target golangci-lint: `{TARGET_GOLANGCI}`",
        "",
        "| Repo | Class | Dirty | Go | golangci | Glazed | Logcopter | Notes |",
        "|---|---|---:|---|---|---:|---:|---|",
    ]
    for a in audits:
        lines.append(
            "| {name} | {classification} | {dirty} | `{go_version}` | `{golangci_version}` | {has_glazed} | {has_logcopter} | {notes} |".format(
                name=a.name,
                classification=a.classification,
                dirty="yes" if a.dirty else "no",
                go_version=a.go_version or "-",
                golangci_version=a.golangci_version or "-",
                has_glazed="yes" if a.has_glazed else "no",
                has_logcopter="yes" if a.has_logcopter else "no",
                notes=a.notes.replace("|", "\\|"),
            )
        )
    lines.append("")
    return "\n".join(lines)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--root",
        type=Path,
        default=Path.home() / "code" / "wesen" / "go-go-golems",
        help="directory containing first-level repositories",
    )
    parser.add_argument("--json", type=Path, help="optional JSON output path")
    parser.add_argument("--markdown", type=Path, help="optional Markdown output path")
    args = parser.parse_args()

    audits = []
    for repo in sorted(args.root.iterdir()):
        if not repo.is_dir():
            continue
        audit = audit_repo(repo)
        if audit is not None:
            audits.append(audit)

    md = markdown_table(audits)
    if args.markdown:
        args.markdown.write_text(md)
    else:
        print(md)
    if args.json:
        args.json.write_text(json.dumps([asdict(a) for a in audits], indent=2) + "\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
