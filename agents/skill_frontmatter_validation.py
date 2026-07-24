"""Validate portable skill frontmatter and discover canonical skills."""

from __future__ import annotations

import re
from pathlib import Path

from skill_validation_paths import CANONICAL_ROOT


PORTABLE_KEYS = {"name", "description"}
CLAUDE_WRAPPER_KEYS = {
    "name",
    "description",
    "argument-hint",
    "disable-model-invocation",
    "user-invocable",
}
CLAUDE_BOOLEAN_KEYS = {"disable-model-invocation", "user-invocable"}
NAME_RE = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")
TOP_LEVEL_KEY_RE = re.compile(r"^([A-Za-z][A-Za-z0-9_-]*):(?:\s*(.*))?$")


def parse_frontmatter(
    path: Path,
) -> tuple[dict[str, str], dict[str, str], list[str]]:
    errors: list[str] = []
    try:
        lines = path.read_text(encoding="utf-8").splitlines()
    except (OSError, UnicodeDecodeError) as exc:
        return {}, {}, [f"{path}: cannot read UTF-8 text: {exc}"]

    if not lines or lines[0] != "---":
        return {}, {}, [f"{path}: frontmatter must start with ---"]
    try:
        end = lines.index("---", 1)
    except ValueError:
        return {}, {}, [f"{path}: frontmatter has no closing ---"]

    fields: dict[str, str] = {}
    current_key: str | None = None
    block_values: dict[str, list[str]] = {}
    for line_number, line in enumerate(lines[1:end], start=2):
        if not line.strip():
            if current_key and fields.get(current_key, "").startswith((">", "|")):
                block_values.setdefault(current_key, []).append("")
            continue
        if line[0].isspace():
            if current_key is None:
                errors.append(f"{path}:{line_number}: unexpected indentation")
            elif not fields.get(current_key, "").startswith((">", "|")):
                errors.append(
                    f"{path}:{line_number}: indented continuation requires a block scalar"
                )
            else:
                block_values.setdefault(current_key, []).append(line.strip())
            continue

        match = TOP_LEVEL_KEY_RE.match(line)
        if not match:
            errors.append(f"{path}:{line_number}: invalid frontmatter line")
            current_key = None
            continue
        key, raw_value = match.group(1), (match.group(2) or "").strip()
        if key in fields:
            errors.append(f"{path}:{line_number}: duplicate frontmatter key {key}")
        fields[key] = raw_value
        current_key = key
        if (
            raw_value
            and not raw_value.startswith(("'", '"', ">", "|"))
            and ": " in raw_value
        ):
            errors.append(
                f"{path}:{line_number}: quote or fold a scalar containing ': '"
            )

    raw_fields = fields.copy()
    for key, values in block_values.items():
        if fields.get(key, "").startswith(">"):
            fields[key] = " ".join(values)
        elif fields.get(key, "").startswith("|"):
            fields[key] = "\n".join(values)

    for key, value in list(fields.items()):
        if len(value) >= 2 and value[0] == value[-1] and value[0] in {'"', "'"}:
            fields[key] = value[1:-1]
    return fields, raw_fields, errors


def validate_skill_file(
    skill_dir: Path, allowed_keys: set[str], label: str
) -> list[str]:
    path = skill_dir / "SKILL.md"
    if not path.is_file():
        return [f"{skill_dir}: missing SKILL.md"]

    fields, raw_fields, errors = parse_frontmatter(path)
    unexpected = set(fields) - allowed_keys
    missing = {"name", "description"} - set(fields)
    if unexpected:
        errors.append(f"{path}: unsupported {label} keys: {sorted(unexpected)}")
    if missing:
        errors.append(f"{path}: missing required keys: {sorted(missing)}")

    name = fields.get("name", "")
    if raw_fields.get("name") != name:
        errors.append(f"{path}: name must be an unquoted plain scalar")
    if name != skill_dir.name:
        errors.append(f"{path}: name {name!r} must equal directory {skill_dir.name!r}")
    if name and (len(name) > 64 or not NAME_RE.fullmatch(name)):
        errors.append(f"{path}: invalid skill name {name!r}")

    description = fields.get("description", "")
    if raw_fields.get("description") not in {">", ">-", "|", "|-"}:
        errors.append(f"{path}: description must use a YAML block scalar")
    if not description:
        errors.append(f"{path}: description must not be empty")
    else:
        if "<" in description or ">" in description:
            errors.append(f"{path}: description must not contain angle brackets")
        if len(description) > 1024:
            errors.append(
                f"{path}: description is {len(description)} characters; max is 1024"
            )

    if "argument-hint" in raw_fields:
        raw_hint = raw_fields["argument-hint"]
        if not (len(raw_hint) >= 2 and raw_hint[0] == raw_hint[-1] == '"'):
            errors.append(f"{path}: argument-hint must be a double-quoted string")
    for key in CLAUDE_BOOLEAN_KEYS:
        if key in raw_fields and raw_fields[key] not in {"true", "false"}:
            errors.append(f"{path}: {key} must be true or false")
    return errors


def canonical_skills() -> dict[str, Path]:
    if not CANONICAL_ROOT.is_dir():
        return {}
    return {
        child.name: child
        for child in sorted(CANONICAL_ROOT.iterdir(), key=lambda path: path.name)
        if child.is_dir() and not child.name.startswith(".")
    }
