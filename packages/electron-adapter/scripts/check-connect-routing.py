#!/usr/bin/env python3
"""Static guard: every desktop renderer Connect call must derive a real proto path.

Why this exists: the desktop renderer has no wasm, so `getXxxService().<verb>Connect`
calls fall through `withConnectFallback` (packages/electron-adapter/src/connect-fallback.ts),
which derives a backend `(service, method)` by string-munging the verb plus a few
hand-maintained override maps. When a wasm verb diverges from its proto RPC name
(suffix/prefix/sibling-service) and nobody updates the map, the derived path is a
VALID-looking name that the backend never registered → silent desktop-only 404
(web is unaffected; it routes through the wasm api-client). #429-era billing was
exactly this: 6 verbs 404'd because the facade had no override map.

This catches that class even when a unit test for the specific verb is missing:
it greps the live renderer call sites, replays the EXACT derivation against the
real wiring, and asserts each result against the authoritative proto paths the
Rust api-client encodes — failing closed instead of shipping a 404.

Run:  python3 packages/electron-adapter/scripts/check-connect-routing.py
Exits non-zero (and lists the offenders) on any unrouteable or unknown call.
"""
from __future__ import annotations  # PEP 604 `X | None` annotations on Python <3.10

import os
import re
import subprocess
import sys

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "..", "..", ".."))
ADAPTER = "packages/electron-adapter/src"


def sh(cmd: str) -> str:
    r = subprocess.run(cmd, shell=True, cwd=REPO_ROOT, capture_output=True, text=True)
    # grep exits 1 on "no matches" (legitimate) but >1 on real errors (bad path /
    # bad regex). Fail loud on the latter — a swallowed error would silently empty
    # the result set and turn this guard into a green no-op.
    if r.returncode > 1:
        sys.exit(f"check-connect-routing: command errored ({r.returncode}): {cmd}\n{r.stderr.strip()}")
    return r.stdout


def authoritative_paths() -> set[str]:
    """The proto paths the Rust api-client actually targets — the backend SSOT."""
    out = sh(
        "grep -rhoE '\"/proto\\.[a-z_]+\\.v[0-9]+\\.[A-Za-z]+/[A-Za-z0-9]+\"' "
        "clients/core/crates/api-client/src/modules/*.rs"
    )
    return {line.strip().strip('"').lower() for line in out.splitlines()}


def override_maps() -> dict[str, dict[str, str]]:
    src = open(os.path.join(REPO_ROOT, ADAPTER, "connect-fallback.ts")).read()
    maps: dict[str, dict[str, str]] = {}
    for m in re.finditer(r"export const (\w+):[^=]*=\s*\{(.*?)\};", src, re.S):
        maps[m.group(1)] = {
            kv.group(1): kv.group(2)
            for kv in re.finditer(r'(\w+):\s*"([^"]+)"', m.group(2))
        }
    return maps


def getter_to_key() -> dict[str, str]:
    src = open(os.path.join(REPO_ROOT, "packages/service-runtime/src/service-getters.ts")).read()
    return {
        m.group(1): m.group(2)
        for m in re.finditer(r'export const (get\w+) = \(\)[^=]*=>\s*g\("(\w+)"\)', src)
    }


def provider_wiring() -> tuple[dict, dict]:
    src = open(os.path.join(REPO_ROOT, ADAPTER, "provider.ts")).read()
    cls2file = {
        m.group(1): m.group(2)[2:]
        for m in re.finditer(r"import\s*\{\s*(Electron\w+)\s*\}\s*from\s*'(\./[\w_]+)'", src)
    }
    var2cls = {m.group(1): m.group(2) for m in re.finditer(r"const (\w+) = new (Electron\w+)\(\)", src)}
    wiring: dict[str, dict] = {}
    for m in re.finditer(
        r"(\w+):\s*withConnectFallback\(\s*(new (Electron\w+)\(\)|\w+)\s*,\s*\"([^\"]+)\""
        r"\s*(?:,\s*(\w+|undefined))?\s*(?:,\s*(\w+))?\s*,?\s*\)",
        src, re.S,
    ):
        key, arg1, cls_inline, proto, sov, nov = m.groups()
        wiring[key] = dict(
            proto=proto,
            sov=None if sov in (None, "undefined") else sov,
            nov=nov,
            cls=cls_inline or var2cls.get(arg1),
        )
    return wiring, cls2file


def facade_methods(cls: str | None, cls2file: dict[str, str]) -> set[str]:
    """Methods a facade implements directly — these bypass the proxy entirely."""
    f = cls2file.get(cls or "")
    if not f:
        return set()
    p = os.path.join(REPO_ROOT, ADAPTER, f"{f}.ts")
    if not os.path.exists(p):
        return set()
    # Heuristic, not a parser: capture indented `(?:async) name(` declarations.
    # Anchoring to line-start excludes keyword-prefixed call sites (`return foo(`,
    # `await bar(`, `if (`) — enough for these facades, whose class bodies hold only
    # declarations. A bare indented `verbConnect(x)` statement would still be captured;
    # acceptable because no facade contains one, and a wrong bypass only risks a missed
    # proxy-route check that the in-language connect-fallback.test.ts also guards.
    return set(re.findall(r"^[ \t]+(?:async[ \t]+)?([a-zA-Z_]\w*)[ \t]*\(", open(p).read(), re.M))


def renderer_calls() -> list[str]:
    out = sh(
        "grep -rhoE 'get[A-Za-z]+Service\\(\\)\\.[a-zA-Z_]+(Connect|_connect)' "
        "clients/web/src clients/desktop/src"
    )
    return sorted(set(out.split()))


def pascal(s: str) -> str:
    return s[:1].upper() + s[1:]


def main() -> int:
    auth = authoritative_paths()
    maps = override_maps()
    g2k = getter_to_key()
    wiring, cls2file = provider_wiring()
    calls = renderer_calls()

    # Fail closed: empty inputs mean the source tree moved or the script ran from
    # the wrong place — NOT "all clean". A guard that checks nothing must never
    # report OK. (renderer_calls() only sees single-line `getX().vConnect` calls;
    # verbs reached via a stored variable or multi-line chain are a known blind
    # spot — see the grep in renderer_calls().)
    if not auth:
        sys.exit("check-connect-routing: 0 authoritative proto paths found — api-client moved or wrong cwd?")
    if not calls:
        sys.exit("check-connect-routing: 0 renderer Connect calls found — source paths moved or wrong cwd?")

    bugs: list[str] = []
    unknown: list[str] = []
    bypassed = 0
    seen: set[tuple[str, str]] = set()

    for call in calls:
        mm = re.match(r"(get[A-Za-z]+Service)\(\)\.([a-zA-Z_]+?)(Connect|_connect)$", call)
        if not mm:
            continue
        getter, stem, suf = mm.groups()
        prop = stem + suf
        if (getter, prop) in seen:
            continue
        seen.add((getter, prop))

        key = g2k.get(getter)
        if not key:
            unknown.append(f"{call}  [getter not in service-getters.ts]")
            continue
        if key not in wiring:
            unknown.append(f"{call}  [key '{key}' not wired via withConnectFallback]")
            continue
        w = wiring[key]
        if prop in facade_methods(w["cls"], cls2file):
            bypassed += 1
            continue

        if suf == "Connect":
            camel = stem
        else:
            camel = re.sub(r"_([a-z0-9])", lambda x: x.group(1).upper(), stem)
        method = maps.get(w["nov"], {}).get(camel, pascal(camel))
        service = maps.get(w["sov"], {}).get(camel, w["proto"])
        if f"/{service}/{method}".lower() not in auth:
            bugs.append(f"{getter}().{prop}  ->  {service}/{method}   [NOT A REGISTERED PROTO PATH]")

    print(
        f"checked {len(seen)} renderer Connect calls "
        f"({bypassed} handled directly by a facade, {len(seen) - bypassed} via proxy); "
        f"{len(auth)} authoritative proto paths."
    )
    if bugs:
        print(f"\nFAIL: {len(bugs)} call(s) derive a path the backend never registered (desktop-only 404):")
        for b in bugs:
            print("  ✗", b)
    if unknown:
        print(f"\nFAIL: {len(unknown)} call(s) hit an unrecognized facade:")
        for u in unknown:
            print("  ?", u)
    if bugs or unknown:
        print("\nFix: add the missing route to the override maps in "
              f"{ADAPTER}/connect-fallback.ts (service via methodOverrides, name via methodNameOverrides).")
        return 1
    print("OK: every desktop Connect call routes to a registered proto path.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
