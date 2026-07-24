"""Validate canonical skill content and bundled resource references."""

from __future__ import annotations

import re
from pathlib import Path


MARKDOWN_LINK_RE = re.compile(r"!?\[[^\]]*\]\(([^)]+)\)")
BUNDLED_PATH_RE = re.compile(
    r"\x60((?:scripts|references|assets|templates|examples|checklists|data)/"
    r"[^\x60\s]+)\x60"
)
PORTABILITY_PATTERNS = {
    "Claude project path": re.compile(r"\.claude/"),
    "Codex project path": re.compile(r"\.codex/"),
    "Codex discovery path": re.compile(r"\.agents/skills/"),
    "Claude argument substitution": re.compile(r"\$ARGUMENTS\b"),
    "Claude environment substitution": re.compile(r"\$\{?CLAUDE_[A-Z0-9_]+\}?"),
    "Codex environment substitution": re.compile(r"\$\{?CODEX_[A-Z0-9_]+\}?"),
    "Claude interaction tool": re.compile(r"\bAskUserQuestion\b"),
    "client-specific task tool": re.compile(
        r"\b(?:TaskCreate|TaskGet|TaskList|TaskOutput|TaskStop|TaskUpdate|"
        r"TodoWrite|EnterPlanMode|ExitPlanMode|spawn_agent|send_message|"
        r"followup_task|request_user_input|update_plan)\b"
    ),
    "serialized MCP tool name": re.compile(r"\bmcp__[A-Za-z0-9_]+"),
    "personal macOS path": re.compile(r"/Users/[^/\s]+/"),
    "personal Linux path": re.compile(r"/home/[^/\s]+/"),
    "personal Windows path": re.compile(r"[A-Za-z]:\\Users\\[^\\\s]+(?:\\|/)"),
    "provider attribution": re.compile(
        r"(?:Generated with (?:Claude|Codex)|Co-Authored-By:\s*(?:Claude|Codex))",
        re.IGNORECASE,
    ),
    "non-portable wiki link": re.compile(
        r"\[\[[A-Za-z0-9][A-Za-z0-9._/-]*(?:#[^\]]+)?\]\]"
    ),
}


def iter_text_files(skill_dir: Path):
    for path in sorted(skill_dir.rglob("*")):
        if not path.is_file() or path.is_symlink():
            continue
        try:
            data = path.read_bytes()
        except OSError:
            continue
        if b"\0" in data:
            continue
        try:
            yield path, data.decode("utf-8")
        except UnicodeDecodeError:
            continue


def validate_portability(skills: dict[str, Path]) -> list[str]:
    errors: list[str] = []
    for skill_dir in skills.values():
        for path, text in iter_text_files(skill_dir):
            for label, pattern in PORTABILITY_PATTERNS.items():
                if label == "non-portable wiki link" and path.suffix.lower() != ".md":
                    continue
                for match in pattern.finditer(text):
                    line = text.count("\n", 0, match.start()) + 1
                    errors.append(
                        f"{path}:{line}: {label}: {match.group(0)!r}"
                    )
    return errors


def validate_markdown_links(skills: dict[str, Path]) -> list[str]:
    errors: list[str] = []
    for skill_dir in skills.values():
        for path in skill_dir.rglob("*.md"):
            text = path.read_text(encoding="utf-8")
            for match in MARKDOWN_LINK_RE.finditer(text):
                raw_target = match.group(1).strip().split()[0].strip("<>")
                target = raw_target.split("#", 1)[0]
                if not target or "://" in target or target.startswith(("#", "mailto:")):
                    continue
                if any(marker in target for marker in ("{", "}", "$", "*")):
                    continue
                candidate = (path.parent / target).resolve()
                if not candidate.exists():
                    line = text.count("\n", 0, match.start()) + 1
                    errors.append(
                        f"{path}:{line}: missing Markdown target {raw_target!r}"
                    )
    return errors


def validate_bundled_paths(skills: dict[str, Path]) -> list[str]:
    errors: list[str] = []
    for skill_dir in skills.values():
        for path in skill_dir.rglob("*.md"):
            text = path.read_text(encoding="utf-8")
            for match in BUNDLED_PATH_RE.finditer(text):
                target = match.group(1).rstrip(".,:;")
                if any(marker in target for marker in ("{", "}", "<", ">", "$", "*")):
                    continue
                if not (skill_dir / target).exists():
                    line = text.count("\n", 0, match.start()) + 1
                    errors.append(
                        f"{path}:{line}: missing bundled path {target!r}"
                    )
    return errors
