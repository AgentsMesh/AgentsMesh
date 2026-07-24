#!/usr/bin/env python3
"""Regression tests for the repository skill validator."""

from __future__ import annotations

import tempfile
import unittest
from pathlib import Path
from unittest import mock

import skill_content_validation as content_validation
import skill_frontmatter_validation as frontmatter_validation
import skill_projection_validation as projection_validation


VALID_BODY = """---
name: {name}
description: >-
  Performs a focused repository workflow. Use when this workflow is requested.
---

# Workflow
"""


class SkillValidatorTests(unittest.TestCase):
    def make_skill(self, root: Path, name: str, content: str | None = None) -> Path:
        skill = root / name
        skill.mkdir(parents=True)
        if content is not None:
            (skill / "SKILL.md").write_text(content, encoding="utf-8")
        return skill

    def test_accepts_portable_frontmatter(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            skill = self.make_skill(
                Path(temporary), "sample-skill", VALID_BODY.format(name="sample-skill")
            )
            self.assertEqual(
                frontmatter_validation.validate_skill_file(
                    skill, frontmatter_validation.PORTABLE_KEYS, "portable"
                ),
                [],
            )

    def test_rejects_indented_plain_scalar_continuation(self) -> None:
        content = VALID_BODY.format(name="sample-skill").replace(
            "name: sample-skill", "name: sample-skill\n  hidden-suffix"
        )
        with tempfile.TemporaryDirectory() as temporary:
            skill = self.make_skill(Path(temporary), "sample-skill", content)
            errors = frontmatter_validation.validate_skill_file(
                skill, frontmatter_validation.PORTABLE_KEYS, "portable"
            )
            self.assertTrue(
                any("indented continuation requires a block scalar" in error for error in errors)
            )

    def test_rejects_non_scalar_description(self) -> None:
        content = VALID_BODY.format(name="sample-skill").replace(
            "description: >-\n  Performs a focused repository workflow. Use when this workflow is requested.",
            "description: []",
        )
        with tempfile.TemporaryDirectory() as temporary:
            skill = self.make_skill(Path(temporary), "sample-skill", content)
            errors = frontmatter_validation.validate_skill_file(
                skill, frontmatter_validation.PORTABLE_KEYS, "portable"
            )
            self.assertTrue(any("must use a YAML block scalar" in error for error in errors))

    def test_rejects_angle_brackets_in_description(self) -> None:
        content = VALID_BODY.format(name="sample-skill").replace(
            "Performs a focused repository workflow.",
            "Runs an existing .iterations/<name> workflow.",
        )
        with tempfile.TemporaryDirectory() as temporary:
            skill = self.make_skill(Path(temporary), "sample-skill", content)
            errors = frontmatter_validation.validate_skill_file(
                skill, frontmatter_validation.PORTABLE_KEYS, "portable"
            )
            self.assertTrue(any("must not contain angle brackets" in error for error in errors))

    def test_catalog_includes_directory_without_skill_file(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            self.make_skill(root, "incomplete-skill")
            with mock.patch.object(
                frontmatter_validation, "CANONICAL_ROOT", root
            ):
                skills = frontmatter_validation.canonical_skills()
            self.assertIn("incomplete-skill", skills)
            self.assertTrue(
                any(
                    "missing SKILL.md" in error
                    for error in frontmatter_validation.validate_skill_file(
                        skills["incomplete-skill"],
                        frontmatter_validation.PORTABLE_KEYS,
                        "portable",
                    )
                )
            )

    def test_checks_missing_markdown_image_target(self) -> None:
        content = VALID_BODY.format(name="sample-skill") + "![preview](assets/missing.png)\n"
        with tempfile.TemporaryDirectory() as temporary:
            skill = self.make_skill(Path(temporary), "sample-skill", content)
            errors = content_validation.validate_markdown_links({skill.name: skill})
            self.assertTrue(any("assets/missing.png" in error for error in errors))

    def test_rejects_client_specific_task_tool(self) -> None:
        content = (
            VALID_BODY.format(name="sample-skill")
            + "Create every plan item with TaskCreate.\n"
        )
        with tempfile.TemporaryDirectory() as temporary:
            skill = self.make_skill(Path(temporary), "sample-skill", content)
            errors = content_validation.validate_portability({skill.name: skill})
            self.assertTrue(
                any("client-specific task tool" in error for error in errors)
            )

    def test_rejects_canonical_resource_symlink(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            skill = self.make_skill(
                root, "sample-skill", VALID_BODY.format(name="sample-skill")
            )
            target = root / "outside.md"
            target.write_text("outside\n", encoding="utf-8")
            (skill / "reference.md").symlink_to(target)
            errors = projection_validation.validate_canonical_links(
                {skill.name: skill}
            )
            self.assertTrue(any("must be real files" in error for error in errors))

    def test_accepts_thin_claude_wrapper(self) -> None:
        wrapper_body = """---
name: sample-skill
description: >-
  Adapts invocation metadata for the shared workflow.
user-invocable: true
---

Follow [the shared workflow](shared/SKILL.md).
"""
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            canonical = self.make_skill(
                root / "canonical",
                "sample-skill",
                VALID_BODY.format(name="sample-skill"),
            )
            wrapper = self.make_skill(
                root / "claude", "sample-skill", wrapper_body
            )
            (wrapper / "shared").symlink_to("../../canonical/sample-skill")
            with mock.patch.object(
                projection_validation, "REPO_ROOT", root.resolve()
            ):
                self.assertEqual(
                    projection_validation.validate_claude_entry(
                        wrapper, canonical
                    ),
                    [],
                )

    def test_rejects_wrapper_that_ignores_shared_workflow(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            canonical = self.make_skill(
                root / "canonical",
                "sample-skill",
                VALID_BODY.format(name="sample-skill"),
            )
            wrapper = self.make_skill(
                root / "claude",
                "sample-skill",
                VALID_BODY.format(name="sample-skill"),
            )
            (wrapper / "shared").symlink_to("../../canonical/sample-skill")
            with mock.patch.object(
                projection_validation, "REPO_ROOT", root.resolve()
            ):
                errors = projection_validation.validate_claude_entry(
                    wrapper, canonical
                )
            self.assertTrue(
                any("must reference shared/SKILL.md" in error for error in errors)
            )

    def test_rejects_invalid_claude_wrapper_boolean(self) -> None:
        content = VALID_BODY.format(name="sample-skill").replace(
            "---\n\n# Workflow",
            "user-invocable: sometimes\n---\n\n# Workflow",
        )
        with tempfile.TemporaryDirectory() as temporary:
            skill = self.make_skill(Path(temporary), "sample-skill", content)
            errors = frontmatter_validation.validate_skill_file(
                skill,
                frontmatter_validation.CLAUDE_WRAPPER_KEYS,
                "Claude wrapper",
            )
            self.assertTrue(
                any("user-invocable must be true or false" in error for error in errors)
            )

    def test_checks_bundled_resource_paths(self) -> None:
        content = (
            VALID_BODY.format(name="sample-skill")
            + "Read `references/contract.md`.\n"
        )
        with tempfile.TemporaryDirectory() as temporary:
            skill = self.make_skill(Path(temporary), "sample-skill", content)
            errors = content_validation.validate_bundled_paths(
                {skill.name: skill}
            )
            self.assertTrue(any("references/contract.md" in error for error in errors))

            references = skill / "references"
            references.mkdir()
            (references / "contract.md").write_text("contract\n", encoding="utf-8")
            self.assertEqual(
                content_validation.validate_bundled_paths({skill.name: skill}),
                [],
            )


if __name__ == "__main__":
    unittest.main()
