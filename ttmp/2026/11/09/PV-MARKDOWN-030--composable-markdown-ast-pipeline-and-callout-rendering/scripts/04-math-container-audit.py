#!/usr/bin/env python3
"""Approximate source audit for multiline display math crossing quote/list scope.

This is a conservative lexical probe, not a Goldmark parser: it skips fenced
code but does not model lazy continuation, every nesting form, or publication
filters. Review reported paths and source before treating them as real bugs.
"""
import argparse
import collections
import os
import re

QUOTE = re.compile(r"^ {0,3}(?:> ?)+")
LIST = re.compile(r"^(\s*)(?:[-+*]|\d+[.)])\s+")
FENCE = re.compile(r"^\s{0,3}(`{3,}|~{3,})")


def quote_depth(line):
    m = QUOTE.match(line)
    return m.group().count(">") if m else 0


def scan(path):
    try:
        with open(path, encoding="utf-8") as f:
            lines = f.readlines()
    except (OSError, UnicodeError):
        return []
    findings = []
    fence = None
    i = 0
    while i < len(lines):
        raw = lines[i].rstrip("\r\n")
        depth = quote_depth(raw)
        after = QUOTE.sub("", raw, count=1) if depth else raw
        m = FENCE.match(after)
        if m:
            marker = m.group(1)
            if fence is None:
                fence = (marker[0], len(marker))
            elif marker[0] == fence[0] and len(marker) >= fence[1]:
                fence = None
            i += 1
            continue
        if fence is not None:
            i += 1
            continue
        list_match = LIST.match(after)
        list_indent = len(list_match.group()) if list_match else None
        content = after[list_match.end():] if list_match else after.lstrip()
        opener = "$$" if content.startswith("$$") else r"\[" if content.startswith(r"\[") else None
        if not opener or (depth == 0 and list_match is None):
            i += 1
            continue
        closer = "$$" if opener == "$$" else r"\]"
        if closer in content[len(opener):]:
            i += 1
            continue
        scope = "quote" if depth else "list"
        j = i + 1
        crossed = False
        while j < len(lines):
            candidate = lines[j].rstrip("\r\n")
            if depth and candidate.strip() and quote_depth(candidate) < depth:
                crossed = True
            if not depth and list_indent is not None and candidate.strip() and len(candidate) - len(candidate.lstrip(" ")) < list_indent:
                crossed = True
            if closer in candidate:
                break
            j += 1
        if j < len(lines):
            findings.append((scope, i + 1, j + 1, crossed))
        # The closer belongs to this pair; treating it as the next opener
        # produces false crossing reports for adjacent valid display blocks.
        i = j + 1 if j < len(lines) else i + 1
    return findings


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("root")
    parser.add_argument("--examples", type=int, default=10)
    args = parser.parse_args()
    totals = collections.Counter()
    docs = collections.defaultdict(set)
    same_scope = collections.Counter()
    same_docs = collections.defaultdict(set)
    examples = []
    scanned = 0
    for base, dirs, files in os.walk(args.root):
        dirs[:] = [d for d in dirs if d not in {".git", "node_modules"}]
        for name in files:
            if not name.lower().endswith(".md"):
                continue
            path = os.path.join(base, name)
            scanned += 1
            for scope, start, end, crossed in scan(path):
                if not crossed:
                    same_scope[scope] += 1
                    same_docs[scope].add(path)
                    continue
                totals[scope] += 1
                docs[scope].add(path)
                if len(examples) < args.examples:
                    examples.append((os.path.relpath(path, args.root), scope, start, end))
    print(f"markdown_files_scanned={scanned}")
    for scope in ("quote", "list"):
        print(f"{scope}: same_scope_pairs={same_scope[scope]} same_scope_documents={len(same_docs[scope])} crossing_candidates={totals[scope]} crossing_documents={len(docs[scope])}")
    for path, scope, start, end in examples:
        print(f"candidate {scope} {path}:{start} closer_line={end}")

if __name__ == "__main__":
    main()
