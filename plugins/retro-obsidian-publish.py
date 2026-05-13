#!/usr/bin/env python3
import json
import os
import shutil
import sys
from pathlib import Path


def emit(obj):
    sys.stdout.write(json.dumps(obj, separators=(",", ":")) + "\n")
    sys.stdout.flush()


def repo_root(ctx):
    return Path(ctx.get("repo_root") or os.getcwd()).resolve()


emit({
    "type": "handshake",
    "protocol_version": "v2",
    "plugin_name": "retro-obsidian-publish",
    "capabilities": {"ops": ["config.mutate", "validate.run", "launch.plan"]},
})


def handle_config(rid, ctx):
    vault = os.environ.get("VAULT_DIR", "backend/vault-example")
    emit({
        "type": "response",
        "request_id": rid,
        "ok": True,
        "output": {"config_patch": {"set": {
            "env.vault_dir": vault,
            "paths.backend": "backend",
            "paths.web": "web",
            "services.backend.port": 8080,
            "services.backend.url": "http://127.0.0.1:8080",
            "services.web.port": 3000,
            "services.web.url": "http://127.0.0.1:3000",
            "services.web.api_url": "http://127.0.0.1:8080",
        }, "unset": []}},
    })


def handle_validate(rid, ctx):
    root = repo_root(ctx)
    errors = []
    warnings = []

    for exe in ["go", "node", "pnpm"]:
        if shutil.which(exe) is None:
            errors.append({"code": "E_MISSING_TOOL", "message": f"{exe} not found on PATH"})

    checks = [
        (root / "backend" / "go.mod", "E_MISSING_BACKEND", "backend/go.mod not found"),
        (root / "backend" / "cmd" / "retro-obsidian-publish" / "main.go", "E_MISSING_CLI", "single-binary CLI entrypoint not found"),
        (root / "web" / "package.json", "E_MISSING_WEB", "web/package.json not found"),
    ]
    for path, code, message in checks:
        if not path.exists():
            errors.append({"code": code, "message": message})

    if not (root / "web" / "node_modules").exists():
        warnings.append({"code": "W_WEB_DEPS", "message": "web/node_modules missing; run pnpm --dir web install --frozen-lockfile"})
    if not (root / "web" / "dist" / "index.html").exists():
        warnings.append({"code": "W_WEB_DIST", "message": "web/dist missing; run cd backend && go run ./cmd/retro-obsidian-publish build web"})
    if not (root / "backend" / "vault-example").exists():
        warnings.append({"code": "W_NO_EXAMPLE_VAULT", "message": "backend/vault-example is missing"})

    emit({
        "type": "response",
        "request_id": rid,
        "ok": True,
        "output": {"valid": len(errors) == 0, "errors": errors, "warnings": warnings},
    })


def handle_launch(rid, ctx):
    emit({
        "type": "response",
        "request_id": rid,
        "ok": True,
        "output": {"services": [
            {
                "name": "backend",
                "cwd": "backend",
                "command": ["go", "run", "./cmd/retro-obsidian-publish", "serve", "--vault", "./vault-example", "--port", "8080", "--serve-web=false"],
                "env": {},
                "health": {"type": "http", "url": "http://127.0.0.1:8080/api/notes", "timeout_ms": 30000},
            },
            {
                "name": "web",
                "cwd": "web",
                "command": ["pnpm", "dev", "--host", "127.0.0.1", "--port", "3000"],
                "env": {"VITE_API_URL": "http://127.0.0.1:8080"},
                "health": {"type": "http", "url": "http://127.0.0.1:3000", "timeout_ms": 30000},
            },
        ]}},
    )


for line in sys.stdin:
    if not line.strip():
        continue
    req = json.loads(line)
    rid = req.get("request_id", "")
    op = req.get("op", "")
    ctx = req.get("ctx", {}) or {}
    try:
        if op == "config.mutate":
            handle_config(rid, ctx)
        elif op == "validate.run":
            handle_validate(rid, ctx)
        elif op == "launch.plan":
            handle_launch(rid, ctx)
        else:
            emit({"type": "response", "request_id": rid, "ok": False,
                  "error": {"code": "E_UNSUPPORTED", "message": f"unsupported op: {op}"}})
    except Exception as e:
        emit({"type": "response", "request_id": rid, "ok": False,
              "error": {"code": "E_PLUGIN", "message": str(e)}})
