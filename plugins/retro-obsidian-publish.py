#!/usr/bin/env python3
import json
import os
import shutil
import socket
import sys
from pathlib import Path


def emit(obj):
    sys.stdout.write(json.dumps(obj, separators=(",", ":")) + "\n")
    sys.stdout.flush()


def log(msg):
    sys.stderr.write(msg + "\n")
    sys.stderr.flush()


def repo_root(ctx):
    return Path(ctx.get("repo_root") or os.getcwd()).resolve()


def port_in_use(port, host="127.0.0.1"):
    """Return True if something is already listening on host:port."""
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.settimeout(0.3)
        return s.connect_ex((host, port)) == 0


def find_free_port(preferred, host="127.0.0.1", max_tries=100):
    """Return the first free port starting from *preferred*.

    If *preferred* is free it is returned immediately.  Otherwise the
    search increments until a free port is found or *max_tries* is
    exhausted (raises RuntimeError).
    """
    for port in range(preferred, preferred + max_tries):
        if not port_in_use(port, host):
            if port != preferred:
                log(f"port {preferred} in use, using {port}")
            return port
    raise RuntimeError(f"no free port found in range {preferred}-{preferred + max_tries - 1}")


# ── Cached ports (set by handle_config, read by handle_launch) ──
_resolved_ports = {}  # {"backend": int, "web": int}


emit({
    "type": "handshake",
    "protocol_version": "v2",
    "plugin_name": "retro-obsidian-publish",
    "capabilities": {"ops": ["config.mutate", "validate.run", "launch.plan"]},
})


def handle_config(rid, ctx):
    vault = os.environ.get("VAULT_DIR", "vault-example")
    vault_name = os.environ.get("VAULT_NAME", "")

    # Resolve ports — env overrides take precedence, then probe from defaults
    backend_port = find_free_port(
        int(os.environ.get("BACKEND_PORT", "8080"))
    )
    web_port = find_free_port(
        int(os.environ.get("WEB_PORT", "3000"))
    )

    # Cache for handle_launch
    _resolved_ports["backend"] = backend_port
    _resolved_ports["web"] = web_port
    ssr_port = find_free_port(
        int(os.environ.get("SSR_PORT", "8089"))
    )
    _resolved_ports["ssr"] = ssr_port

    config_set = {
        "env.vault_dir": vault,
        "paths.backend": ".",
        "paths.web": "web",
        "services.backend.port": backend_port,
        "services.backend.url": f"http://127.0.0.1:{backend_port}",
        "services.web.port": web_port,
        "services.web.url": f"http://127.0.0.1:{web_port}",
        "services.web.api_url": f"http://127.0.0.1:{backend_port}",
        "services.ssr.port": ssr_port,
        "services.ssr.url": f"http://127.0.0.1:{ssr_port}",
    }
    if vault_name:
        config_set["env.vault_name"] = vault_name

    if backend_port != 8080:
        config_set["env.backend_port_shifted"] = "true"
    if web_port != 3000:
        config_set["env.web_port_shifted"] = "true"

    emit({
        "type": "response",
        "request_id": rid,
        "ok": True,
        "output": {"config_patch": {"set": config_set, "unset": []}},
    })


def handle_validate(rid, ctx):
    root = repo_root(ctx)
    errors = []
    warnings = []

    for exe in ["go", "node", "pnpm"]:
        if shutil.which(exe) is None:
            errors.append({"code": "E_MISSING_TOOL", "message": f"{exe} not found on PATH"})

    checks = [
        (root / "go.mod", "E_MISSING_BACKEND", "go.mod not found"),
        (root / "cmd" / "retro-obsidian-publish" / "main.go", "E_MISSING_CLI", "single-binary CLI entrypoint not found"),
        (root / "web" / "package.json", "E_MISSING_WEB", "web/package.json not found"),
    ]
    for path, code, message in checks:
        if not path.exists():
            errors.append({"code": code, "message": message})

    if not (root / "web" / "node_modules").exists():
        warnings.append({"code": "W_WEB_DEPS", "message": "web/node_modules missing; run pnpm --dir web install --frozen-lockfile"})
    if not (root / "web" / "dist" / "index.html").exists():
        warnings.append({"code": "W_WEB_DIST", "message": "web/dist missing; devctl ssr launch will run pnpm build:all before node server.mjs"})

    # Validate vault directory exists
    vault_dir = os.environ.get("VAULT_DIR", "vault-example")
    vault_path = Path(vault_dir)
    if not vault_path.is_absolute():
        vault_path = root / vault_path
    if not vault_path.exists():
        errors.append({"code": "E_VAULT_MISSING", "message": f"vault directory not found: {vault_dir}"})
    elif not any(vault_path.glob("*.md")):
        warnings.append({"code": "W_VAULT_EMPTY", "message": f"vault directory has no .md files: {vault_dir}"})

    emit({
        "type": "response",
        "request_id": rid,
        "ok": True,
        "output": {"valid": len(errors) == 0, "errors": errors, "warnings": warnings},
    })


def handle_launch(rid, ctx):
    vault = os.environ.get("VAULT_DIR", "vault-example")
    vault_name = os.environ.get("VAULT_NAME", "")
    page_title = os.environ.get("PAGE_TITLE", "")

    # Resolve vault path relative to repo root for the backend command
    root = repo_root(ctx)
    vault_path = Path(vault)
    if not vault_path.is_absolute():
        vault_arg = str(root / vault_path)
    else:
        vault_arg = vault

    # Use ports resolved by handle_config (fall back to defaults if
    # handle_config was never called — shouldn't happen in practice)
    backend_port = _resolved_ports.get("backend", 8080)
    web_port = _resolved_ports.get("web", 3000)
    ssr_port = find_free_port(
        int(os.environ.get("SSR_PORT", "8089"))
    )

    backend_cmd = ["go", "run", "./cmd/retro-obsidian-publish", "serve",
                   "--vault", vault_arg,
                   "--port", str(backend_port),
                   "--serve-web=true",
                   "--ssr-url", f"http://127.0.0.1:{ssr_port}"]
    if vault_name:
        backend_cmd.extend(["--vault-name", vault_name])
    if page_title:
        backend_cmd.extend(["--page-title", page_title])

    # Larger vaults (absolute paths, e.g. go-go-parc) need more time for the initial load
    health_timeout = 60000 if vault_path.is_absolute() else 30000

    emit({
        "type": "response",
        "request_id": rid,
        "ok": True,
        "output": {"services": [
            {
                "name": "backend",
                "cwd": ".",
                "command": backend_cmd,
                "env": {"GOWORK": "off"},
                "health": {"type": "http", "url": f"http://127.0.0.1:{backend_port}/api/notes", "timeout_ms": health_timeout},
            },
            {
                "name": "web",
                "cwd": "web",
                "command": ["pnpm", "dev", "--host", "127.0.0.1", "--port", str(web_port)],
                "env": {"VITE_API_URL": f"http://127.0.0.1:{backend_port}"},
                "health": {"type": "http", "url": f"http://127.0.0.1:{web_port}", "timeout_ms": 30000},
            },
            {
                "name": "ssr",
                "cwd": "web",
                "command": ["sh", "-c", "pnpm build:all && exec node server.mjs"],
                "env": {
                    "SSR_PORT": str(ssr_port),
                    "API_BASE": f"http://127.0.0.1:{backend_port}",
                    "BASE_URL": f"http://127.0.0.1:{backend_port}",
                },
                "health": {"type": "http", "url": f"http://127.0.0.1:{ssr_port}/health", "timeout_ms": 30000},
                "depends_on": ["backend"],
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
