#!/usr/bin/env python3
"""Validate portable skills and their Codex/Claude discovery projections."""

import sys

from skill_content_validation import (
    validate_bundled_paths,
    validate_markdown_links,
    validate_portability,
)
from skill_frontmatter_validation import (
    PORTABLE_KEYS,
    canonical_skills,
    validate_skill_file,
)
from skill_projection_validation import (
    validate_canonical_links,
    validate_projections,
)
from skill_validation_paths import CANONICAL_ROOT


def main() -> int:
    skills = canonical_skills()
    errors: list[str] = []
    if not skills:
        errors.append(f"{CANONICAL_ROOT}: no canonical skills found")
    for skill_dir in skills.values():
        if skill_dir.is_symlink():
            errors.append(f"{skill_dir}: canonical skill must be a real directory")
        errors.extend(validate_skill_file(skill_dir, PORTABLE_KEYS, "portable"))
    errors.extend(validate_projections(skills))
    errors.extend(validate_canonical_links(skills))
    errors.extend(validate_portability(skills))
    errors.extend(validate_markdown_links(skills))
    errors.extend(validate_bundled_paths(skills))

    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        print(
            f"Skill validation failed with {len(errors)} error(s).",
            file=sys.stderr,
        )
        return 1
    print(f"Validated {len(skills)} portable skills and both discovery projections.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
