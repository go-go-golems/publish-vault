#!/usr/bin/env python3
import json
import subprocess
import sys
import unittest
from pathlib import Path


class DevctlPluginSmokeTest(unittest.TestCase):
    def test_handshake_is_first_stdout_frame(self):
        plugin = Path(__file__).with_name("retro-obsidian-publish.py")
        proc = subprocess.Popen(
            [sys.executable, str(plugin)],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )
        try:
            line = proc.stdout.readline()
            self.assertTrue(line.strip(), "plugin did not emit handshake")
            frame = json.loads(line)
            self.assertEqual(frame["type"], "handshake")
            self.assertEqual(frame["protocol_version"], "v2")
            self.assertIn("config.mutate", frame["capabilities"]["ops"])
        finally:
            proc.terminate()
            try:
                proc.wait(timeout=2)
            except subprocess.TimeoutExpired:
                proc.kill()
                proc.wait(timeout=2)
            if proc.stdin:
                proc.stdin.close()
            if proc.stdout:
                proc.stdout.close()
            if proc.stderr:
                proc.stderr.close()


if __name__ == "__main__":
    unittest.main()
