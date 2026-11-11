#!/usr/bin/env python3
import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name("01-group-warnings.py")


class GroupWarningsTest(unittest.TestCase):
    def test_grouping_preserves_occurrences_and_protects_output(self):
        rows = [
            dict(rule="WIKI_TARGET_MISSING", target='-x "$(command -v git)"', path="b.md", line=3, column=4),
            dict(rule="WIKI_TARGET_MISSING", target='-x "$(command -v git)"', path="a.md", line=2, column=1),
            dict(rule="WIKI_HEADING_MISSING", target="Note#^block", path="a.md", line=5, column=2),
            dict(rule="WIKI_TARGET_MISSING", target="Absent", path="a.md", line=6, column=1),
        ]
        with tempfile.TemporaryDirectory() as d:
            inp, out = Path(d)/"in.jsonl", Path(d)/"out.json"
            inp.write_text("".join(json.dumps(row)+"\n" for row in rows))
            subprocess.run([sys.executable, str(SCRIPT), str(inp), str(out)], check=True, capture_output=True)
            result = json.loads(out.read_text())
            self.assertEqual(result["total_occurrences"], 4)
            shell = next(g for g in result["groups"] if g["category"] == "possible_shell_test")
            self.assertEqual(shell["count"], 2)
            self.assertEqual([p["path"] for p in shell["occurrences"]], ["a.md", "b.md"])
            self.assertEqual(os.stat(out).st_mode & 0o777, 0o600)
            failed = subprocess.run([sys.executable, str(SCRIPT), str(inp), str(out)], capture_output=True)
            self.assertNotEqual(failed.returncode, 0)
            self.assertEqual(json.loads(out.read_text()), result)


if __name__ == "__main__":
    unittest.main()
