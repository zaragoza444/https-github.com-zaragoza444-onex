#!/usr/bin/env python3
"""Automatic VPS deploy for zblockchainsystem.com payment gateway."""
import os
import sys
from pathlib import Path

import paramiko

ROOT = Path(__file__).resolve().parents[1]
HOST = os.environ.get("SSH_HOST", "51.75.64.28")
USER = os.environ.get("SSH_USER", "ubuntu")
DOMAIN = os.environ.get("ONEX_PRODUCTION_DOMAIN", "zblockchainsystem.com")


def load_env() -> None:
    env_path = ROOT / ".env"
    if not env_path.is_file():
        return
    for line in env_path.read_text(encoding="utf-8", errors="replace").splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, _, val = line.partition("=")
        key, val = key.strip(), val.strip().strip('"').strip("'")
        if key and key not in os.environ:
            os.environ[key] = val


def main() -> int:
    load_env()
    password = os.environ.get("SSH_PASS")
    if not password:
        print("Set SSH_PASS (ubuntu VPS password) then re-run:", file=sys.stderr)
        print("  SSH_PASS='...' python3 scripts/auto-deploy-vps.py", file=sys.stderr)
        return 1

    script = f"""
set -euo pipefail
REPO="$HOME/onex"
GITHUB="https://github.com/zaragoza444/https-github.com-zaragoza444-onex.git"
if [ ! -d "$REPO/.git" ]; then git clone "$GITHUB" "$REPO"; fi
cd "$REPO"
git fetch origin main && git reset --hard origin/main
export ONEX_PRODUCTION_DOMAIN={DOMAIN}
export ONEX_STRIPE_SECRET_KEY="${{ONEX_STRIPE_SECRET_KEY:-}}"
export ONEX_STRIPE_PUBLISHABLE_KEY="${{ONEX_STRIPE_PUBLISHABLE_KEY:-}}"
export ONEX_STRIPE_WEBHOOK_SECRET="${{ONEX_STRIPE_WEBHOOK_SECRET:-}}"
if [ -f scripts/fix-all-system.sh ]; then
  bash scripts/fix-all-system.sh
else
  bash scripts/fix-bridge-9338.sh
fi
curl -sf http://127.0.0.1/bridge/payments/status || curl -sf http://127.0.0.1:9338/bridge/payments/status
"""

    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    print(f"Connecting to {USER}@{HOST}...", flush=True)
    client.connect(HOST, username=USER, password=password, timeout=45)

    env_prefix = ""
    for var in ("ONEX_STRIPE_SECRET_KEY", "ONEX_STRIPE_PUBLISHABLE_KEY", "ONEX_STRIPE_WEBHOOK_SECRET"):
        val = os.environ.get(var, "")
        if val:
            env_prefix += f"export {var}={val!r}; "

    stdin, stdout, stderr = client.exec_command(env_prefix + script, get_pty=True)
    out = stdout.read().decode(errors="replace")
    err = stderr.read().decode(errors="replace")
    client.close()
    print(out)
    if err:
        print(err, file=sys.stderr)
    if "enabled" in out and "payment" in out.lower():
        print(f"\nLIVE: https://{DOMAIN}/payments/?page=donate")
        return 0
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
