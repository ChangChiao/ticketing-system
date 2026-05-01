#!/usr/bin/env python3
"""Apply Cloudflare zone settings, cache rules, and rate limit rules.

Required environment variables:
  CLOUDFLARE_API_TOKEN
  CLOUDFLARE_ZONE_ID
  CLOUDFLARE_HOSTNAME

The script is idempotent at the ruleset phase level: it owns the phase entrypoint
rulesets it creates/updates for this application.
"""

from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.request


API_BASE = "https://api.cloudflare.com/client/v4"


def env(name: str) -> str:
    value = os.environ.get(name, "").strip()
    if not value:
        raise SystemExit(f"missing required environment variable: {name}")
    return value


TOKEN = env("CLOUDFLARE_API_TOKEN")
ZONE_ID = env("CLOUDFLARE_ZONE_ID")
HOSTNAME = env("CLOUDFLARE_HOSTNAME")


def request(method: str, path: str, payload: dict | None = None) -> dict:
    body = None
    headers = {
        "Authorization": f"Bearer {TOKEN}",
        "Content-Type": "application/json",
    }
    if payload is not None:
        body = json.dumps(payload).encode("utf-8")

    req = urllib.request.Request(
        f"{API_BASE}{path}",
        data=body,
        headers=headers,
        method=method,
    )
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            data = json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8")
        raise SystemExit(f"Cloudflare API {method} {path} failed: {exc.code} {detail}") from exc

    if not data.get("success"):
        raise SystemExit(f"Cloudflare API {method} {path} failed: {json.dumps(data)}")
    return data


def patch_setting(setting_id: str, value: str | int) -> None:
    request("PATCH", f"/zones/{ZONE_ID}/settings/{setting_id}", {"value": value})
    print(f"updated setting {setting_id}={value}")


def list_rulesets() -> list[dict]:
    return request("GET", f"/zones/{ZONE_ID}/rulesets").get("result", [])


def upsert_phase_ruleset(phase: str, name: str, rules: list[dict]) -> None:
    rulesets = list_rulesets()
    current = next((r for r in rulesets if r.get("phase") == phase and r.get("kind") == "zone"), None)
    payload = {
        "name": name,
        "description": "Managed by scripts/configure_cloudflare.py",
        "kind": "zone",
        "phase": phase,
        "rules": rules,
    }

    if current:
        request("PUT", f"/zones/{ZONE_ID}/rulesets/{current['id']}", payload)
        print(f"updated ruleset {phase}")
    else:
        request("POST", f"/zones/{ZONE_ID}/rulesets", payload)
        print(f"created ruleset {phase}")


def cache_rules() -> list[dict]:
    host = json.dumps(HOSTNAME)
    return [
        {
            "ref": "ticketing-static-cache",
            "description": "Cache Next.js static assets at edge",
            "expression": f'(http.host eq {host} and starts_with(http.request.uri.path, "/_next/static/"))',
            "action": "set_cache_settings",
            "action_parameters": {
                "cache": True,
                "edge_ttl": {"mode": "override_origin", "default": 2592000},
                "browser_ttl": {"mode": "override_origin", "default": 2592000},
            },
        },
        {
            "ref": "ticketing-api-ws-bypass",
            "description": "Bypass cache for API and WebSocket traffic",
            "expression": (
                f'(http.host eq {host} and '
                '(http.request.uri.path eq "/api" or starts_with(http.request.uri.path, "/api/") '
                'or http.request.uri.path eq "/ws" or starts_with(http.request.uri.path, "/ws/")))'
            ),
            "action": "set_cache_settings",
            "action_parameters": {"cache": False},
        },
    ]


def rate_limit_rules() -> list[dict]:
    host = json.dumps(HOSTNAME)
    return [
        {
            "ref": "ticketing-queue-join-rate-limit",
            "description": "Limit queue join bursts per IP",
            "expression": (
                f'(http.host eq {host} and '
                'http.request.uri.path matches "^/api/events/[^/]+/queue/join$")'
            ),
            "action": "block",
            "ratelimit": {
                "characteristics": ["ip.src"],
                "period": 10,
                "requests_per_period": 5,
                "mitigation_timeout": 10,
            },
        },
        {
            "ref": "ticketing-auth-rate-limit",
            "description": "Limit auth endpoint brute force attempts per IP",
            "expression": f'(http.host eq {host} and starts_with(http.request.uri.path, "/api/auth/"))',
            "action": "block",
            "ratelimit": {
                "characteristics": ["ip.src"],
                "period": 60,
                "requests_per_period": 10,
                "mitigation_timeout": 60,
            },
        },
    ]


def main() -> int:
    for setting_id, value in [
        ("ssl", "strict"),
        ("min_tls_version", "1.2"),
        ("websockets", "on"),
        ("http2", "on"),
        ("http3", "on"),
        ("brotli", "on"),
        ("early_hints", "on"),
    ]:
        patch_setting(setting_id, value)

    upsert_phase_ruleset(
        "http_request_cache_settings",
        "Ticketing cache rules",
        cache_rules(),
    )
    upsert_phase_ruleset(
        "http_ratelimit",
        "Ticketing rate limit rules",
        rate_limit_rules(),
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
