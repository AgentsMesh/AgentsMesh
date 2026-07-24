"""Validate Codex and Claude skill projections."""

from __future__ import annotations

import os
from pathlib import Path

from skill_frontmatter_validation import (
    CLAUDE_WRAPPER_KEYS,
    validate_skill_file,
)
from skill_validation_paths import CLAUDE_ROOT, CODEX_ROOT, REPO_ROOT


def projection_names(root: Path) -> set[str]:
    if not root.is_dir():
        return set()
    return {child.name for child in root.iterdir() if not child.name.startswith(".")}


def validate_relative_link(link: Path, expected: Path) -> list[str]:
    if not link.is_symlink():
        return [f"{link}: expected a relative directory symlink"]
    target_text = os.readlink(link)
    errors: list[str] = []
    if os.path.isabs(target_text):
        errors.append(f"{link}: symlink target must be relative, got {target_text}")
    try:
        resolved = link.resolve(strict=True)
    except OSError as exc:
        return errors + [f"{link}: broken symlink: {exc}"]
    if resolved != expected.resolve():
        errors.append(f"{link}: resolves to {resolved}, expected {expected.resolve()}")
    try:
        resolved.relative_to(REPO_ROOT)
    except ValueError:
        errors.append(f"{link}: target leaves the repository")
    return errors


def validate_claude_entry(entry: Path, canonical: Path) -> list[str]:
    if entry.is_symlink():
        return validate_relative_link(entry, canonical)
    if not entry.is_dir():
        return [f"{entry}: expected a symlink or thin wrapper directory"]

    errors: list[str] = []
    wrapper_entries = {child.name for child in entry.iterdir()}
    if extra := wrapper_entries - {"SKILL.md", "shared"}:
        errors.append(
            f"{entry}: thin wrapper has unsupported entries: {sorted(extra)}"
        )
    errors.extend(
        validate_skill_file(entry, CLAUDE_WRAPPER_KEYS, "Claude wrapper")
    )
    errors.extend(validate_relative_link(entry / "shared", canonical))

    wrapper_file = entry / "SKILL.md"
    if wrapper_file.is_file():
        try:
            wrapper_text = wrapper_file.read_text(encoding="utf-8")
        except (OSError, UnicodeDecodeError) as exc:
            errors.append(f"{wrapper_file}: cannot read UTF-8 text: {exc}")
        else:
            if "shared/SKILL.md" not in wrapper_text:
                errors.append(
                    f"{wrapper_file}: thin wrapper must reference shared/SKILL.md"
                )
    return errors


def validate_projections(skills: dict[str, Path]) -> list[str]:
    errors: list[str] = []
    expected_names = set(skills)
    for root, label in ((CODEX_ROOT, "Codex"), (CLAUDE_ROOT, "Claude")):
        actual_names = projection_names(root)
        if missing := expected_names - actual_names:
            errors.append(f"{root}: missing {label} projections: {sorted(missing)}")
        if extra := actual_names - expected_names:
            errors.append(f"{root}: unknown {label} projections: {sorted(extra)}")

    for name, canonical in skills.items():
        errors.extend(validate_relative_link(CODEX_ROOT / name, canonical))
        errors.extend(validate_claude_entry(CLAUDE_ROOT / name, canonical))
    return errors


def validate_canonical_links(skills: dict[str, Path]) -> list[str]:
    errors: list[str] = []
    for skill_dir in skills.values():
        for path in sorted(skill_dir.rglob("*")):
            if path.is_symlink():
                errors.append(
                    f"{path}: canonical skill resources must be real files or directories"
                )
    return errors
