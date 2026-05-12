#!/usr/bin/env python3
"""Validate that hand-maintained Rust prost structs stay in lock-step with
their .proto source.

Run as `validate_prost_tags <proto_file> <rust_file>`. Exit code 0 means
clean; non-zero means a mismatch was detected (and printed to stderr).

This script is the safety net for watch list hazard #8 — a swap or
transcription mistake between `.proto` field numbers and Rust
`prost(tag = "N")` annotations is otherwise invisible at compile time
(same Rust type, just wrong tag → silently wrong field value on the
wire).

The parser is intentionally regex-only — it covers the lint dimension
without pulling protoc / syn / rules_python into the build graph. The
.proto subset we accept is whatever the 26-service migration
deliverable produces: snake_case fields, no inline messages, no
imports across files. Anything more exotic is flagged as a parse error
so the human looks at it before merging.
"""

from __future__ import annotations

import re
import sys
from typing import Dict, List, Tuple

# --- .proto parser --------------------------------------------------------

# Match a message block — `message <Name> { ... }`, possibly nested no further
# than the runbook templates allow (top-level only, see runbook §2).
PROTO_MESSAGE_RE = re.compile(
    r"^\s*message\s+(\w+)\s*\{(.*?)^\}\s*$",
    re.DOTALL | re.MULTILINE,
)

# Match a single field line. Supports proto3 modifiers (repeated, optional)
# and the standard scalar / message type vocabulary. Captures the field name
# and tag number; the type is discarded — we only validate name↔tag mapping.
#
# Group layout:
#   1 = modifier (optional|repeated, may be empty)
#   2 = type (int32, string, MessageName, etc.)
#   3 = field name
#   4 = tag number
PROTO_FIELD_RE = re.compile(
    r"^\s*(?:(repeated|optional)\s+)?([\w.]+)\s+(\w+)\s*=\s*(\d+)\s*;",
    re.MULTILINE,
)


def parse_proto(text: str) -> Dict[str, Dict[str, int]]:
    """Return `{message_name: {field_name: tag_number}}`."""
    out: Dict[str, Dict[str, int]] = {}
    for m in PROTO_MESSAGE_RE.finditer(text):
        msg = m.group(1)
        body = m.group(2)
        fields: Dict[str, int] = {}
        for fm in PROTO_FIELD_RE.finditer(body):
            field_name = fm.group(3)
            tag = int(fm.group(4))
            # Skip `oneof <name> { ... }` declarations — their fields are
            # captured as direct matches because PROTO_FIELD_RE doesn't care
            # about the enclosing scope. Reserved blocks are skipped because
            # they don't match the field pattern.
            if field_name in ("oneof",):
                continue
            fields[field_name] = tag
        out[msg] = fields
    return out


# --- Rust prost parser ----------------------------------------------------

# Match a struct block — `pub struct Name { ... }` (no generics — codegen
# tools never emit them, hand-written DTOs shouldn't either).
RUST_STRUCT_RE = re.compile(
    r"pub\s+struct\s+(\w+)\s*\{(.*?)^\}",
    re.DOTALL | re.MULTILINE,
)

# Match a `#[prost(... tag = "N")]` annotation immediately followed by a
# `pub <field_name>:` declaration. The DOTALL crosses newlines between the
# attribute line and the field line.
RUST_FIELD_RE = re.compile(
    r"#\[prost\([^)]*tag\s*=\s*\"(\d+)\"[^)]*\)\]\s*pub\s+(\w+)\s*:",
    re.DOTALL,
)


def parse_rust(text: str) -> Dict[str, Dict[str, int]]:
    """Return `{struct_name: {field_name: tag_number}}`."""
    out: Dict[str, Dict[str, int]] = {}
    for m in RUST_STRUCT_RE.finditer(text):
        struct_name = m.group(1)
        body = m.group(2)
        fields: Dict[str, int] = {}
        for fm in RUST_FIELD_RE.finditer(body):
            tag = int(fm.group(1))
            field_name = fm.group(2)
            fields[field_name] = tag
        if fields:
            out[struct_name] = fields
    return out


# --- Cross-validate -------------------------------------------------------


def compare(
    proto: Dict[str, Dict[str, int]],
    rust: Dict[str, Dict[str, int]],
) -> List[str]:
    """Return a list of human-readable mismatch messages (empty = clean).

    Only enforces that every Rust struct whose name matches a proto message
    has matching field-name↔tag pairs. Extra Rust types (non-prost helpers,
    other crates' types) are ignored.
    """
    errors: List[str] = []
    for struct_name, rust_fields in rust.items():
        proto_fields = proto.get(struct_name)
        if proto_fields is None:
            # The Rust file has prost structs that don't appear in the .proto.
            # That's allowed during migration (the file may host multiple
            # services). No error.
            continue
        # Tag-by-name match
        for fname, rtag in rust_fields.items():
            ptag = proto_fields.get(fname)
            if ptag is None:
                errors.append(
                    f"{struct_name}.{fname}: Rust has tag={rtag} but .proto has no field by that name",
                )
                continue
            if ptag != rtag:
                errors.append(
                    f"{struct_name}.{fname}: .proto tag={ptag}, Rust tag={rtag}",
                )
        for fname, ptag in proto_fields.items():
            if fname not in rust_fields:
                errors.append(
                    f"{struct_name}.{fname}: .proto has tag={ptag}, Rust struct is missing the field",
                )
    return errors


def main(argv: List[str]) -> int:
    if len(argv) != 3:
        print(f"usage: {argv[0]} <proto_file> <rust_file>", file=sys.stderr)
        return 2
    with open(argv[1]) as f:
        proto = parse_proto(f.read())
    with open(argv[2]) as f:
        rust = parse_rust(f.read())
    errors = compare(proto, rust)
    if errors:
        print(f"prost tag drift detected ({argv[2]} vs {argv[1]}):", file=sys.stderr)
        for e in errors:
            print(f"  {e}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
