# Repository Agent Skills

`agents/skills/<name>` is the canonical source for repository skills. Keep each
`SKILL.md` portable across coding agents: use only `name` and `description` in
frontmatter, describe capabilities rather than client tool names, and resolve
bundled resources relative to the skill directory.

Each canonical skill is exposed through both native discovery locations:

```text
.agents/skills/<name> -> ../../agents/skills/<name>
.claude/skills/<name> -> ../../agents/skills/<name>
```

Use those direct links for portable skills. When invocation metadata genuinely
differs, keep the canonical skill as the shared workflow and replace only the
affected client projection with a thin wrapper:

```text
.claude/skills/<name>/
├── SKILL.md
└── shared -> ../../../agents/skills/<name>
```

Wrappers may translate client arguments and invocation metadata, but must not
duplicate the shared workflow. Codex-specific invocation policy belongs in the
canonical skill's `agents/openai.yaml`; Claude-specific policy belongs in its
wrapper or project settings.

Choose names that remain unique across project and user scopes. Same-named
project and personal skills may both appear in client selectors.

Run the repository checks after changing skills or projections:

```bash
./agents/validate.sh
```

For a new portable skill, add its canonical directory and both relative
projection links. Never commit an absolute symlink or a link whose target
leaves the repository.

Committed links require a symlink-capable checkout. On Windows, enable
Developer Mode and set `core.symlinks=true` before checkout; otherwise Git may
materialize link targets as plain text files and neither client will discover
the skills correctly. macOS, Linux, and WSL checkouts normally preserve them.

`AGENTS.md` is intentionally a real, concise file rather than a link to the
larger `CLAUDE.md`. This avoids Codex's project-instruction byte limit and keeps
the repository instructions usable on Windows.
