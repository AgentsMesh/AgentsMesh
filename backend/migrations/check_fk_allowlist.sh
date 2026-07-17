#!/usr/bin/env bash
# Asserts the FK set produced by backend/migrations matches fk_allowlist.txt
# exactly. Enforcing that the list only ever SHRINKS needs the base ref and
# lives in the "FK allowlist may only shrink" CI step -- this script alone
# cannot see that a PR appended a line.
#
# Keyed on the delete rule, not just the columns: a migration flipping an
# existing FK from CASCADE to NO ACTION is the exact defect behind both prod
# incidents (42703, 23503), and column-only keys render it invisible.
#
# Reads pg_catalog rather than information_schema: the latter's key_column_usage
# / constraint_column_usage can only be joined on constraint_name, which is
# unique per-table (not per-schema), so duplicate names and composite keys
# cartesian-product into fabricated rows.
set -euo pipefail
export LC_ALL=C

DB_URL="${1:-${DATABASE_URL:-}}"
if [ -z "$DB_URL" ]; then
    echo "usage: $0 <postgres-url>   (or set DATABASE_URL)" >&2
    exit 2
fi
ALLOWLIST="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/fk_allowlist.txt"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

# ON_ERROR_STOP is load-bearing: in -f mode psql exits 0 after a SQL error, so
# without it a broken query yields an empty result that reads as "every FK was
# dropped" -- and once the allowlist reaches zero, as a vacuous pass.
psql "$DB_URL" -At -v ON_ERROR_STOP=1 -f - <<'SQL' | sort > "$tmp/actual"
SELECT c.conrelid::regclass::text || '.' || a.attname
       || ' -> ' || c.confrelid::regclass::text
       || '  [' || CASE c.confdeltype
            WHEN 'a' THEN 'NO ACTION'
            WHEN 'r' THEN 'RESTRICT'
            WHEN 'c' THEN 'CASCADE'
            WHEN 'n' THEN 'SET NULL'
            WHEN 'd' THEN 'SET DEFAULT'
            ELSE 'UNKNOWN:' || c.confdeltype::text
          END || ']  ' || c.conname
FROM pg_constraint c
JOIN LATERAL unnest(c.conkey) WITH ORDINALITY AS k(attnum, ord) ON true
JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = k.attnum
WHERE c.contype = 'f' AND c.connamespace = 'public'::regnamespace;
SQL

if [ ! -r "$ALLOWLIST" ]; then
    echo "FAIL: cannot read $ALLOWLIST" >&2
    exit 2
fi
# `|| :` absorbs grep's exit 1 when the list is all-comments -- the state this
# contract is aiming for. The readability check above keeps it from also
# absorbing a missing file.
grep -vE '^[[:space:]]*(#|$)' "$ALLOWLIST" | sort > "$tmp/expected" || :

added="$(comm -13 "$tmp/expected" "$tmp/actual")"
removed="$(comm -23 "$tmp/expected" "$tmp/actual")"
status=0

if [ -n "$added" ]; then
    echo "FAIL: foreign keys exist that the allowlist does not list. This system"
    echo "      does not use FKs -- express the parent link as a plain indexed"
    echo "      column and handle parent deletion in the service layer"
    echo "      (CLAUDE.md 外键契约). A changed delete rule also lands here."
    echo "$added" | sed 's/^/    + /'
    status=1
fi

if [ -n "$removed" ]; then
    echo "FAIL: the allowlist lists foreign keys that no longer exist. If you"
    echo "      dropped them, delete these lines -- the list must track reality"
    echo "      so that it can only shrink."
    echo "$removed" | sed 's/^/    - /'
    status=1
fi

if [ "$status" -eq 0 ]; then
    echo "OK: $(wc -l < "$tmp/actual" | tr -d ' ') foreign keys, all allowlisted."
fi
exit "$status"
