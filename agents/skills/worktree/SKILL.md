---
name: worktree
description: >-
  Creates or reuses an isolated AgentsMesh Git worktree from a verified base
  branch, preserves existing changes, and optionally starts the
  worktree-scoped development environment. Use when the user asks for a new
  worktree or isolated work for a feature, fix, investigation, or review.
---

# AgentsMesh Worktree

Create worktrees next to the main checkout under `AgentsMesh-Worktrees`.

## Workflow

1. Determine the requested branch name, base branch (default `main`), and
   whether the task actually needs a running development stack.
2. Inspect repository state and identify the base remote from the user's request,
   the base branch upstream, or its URL:

   ```bash
   git status --short --branch
   git remote -v
   git worktree list
   ```

   Preserve all existing changes. Do not stash, reset, commit, or clean the main
   checkout as a side effect of worktree creation.
3. Fetch the selected base:

   ```bash
   git fetch <base-remote> <base-branch>
   ```

4. Derive a filesystem-safe directory name by replacing `/` in the branch with
   `-`. Use `$(dirname "$repo_root")/AgentsMesh-Worktrees`; do not embed a
   user-specific absolute path.
5. Check whether the branch or path already exists. If the branch is new:

   ```bash
   git worktree add -b <branch-name> \
     <worktree-root>/<directory-name> \
     <base-remote>/<base-branch>
   ```

   If the branch already exists and is not checked out elsewhere, attach it
   without `-b`. Never delete or reset an existing branch to make the command
   succeed.
6. In the new worktree, verify:

   ```bash
   git status --short --branch
   git log --oneline -3
   git merge-base --is-ancestor <base-remote>/<base-branch> HEAD
   ```

7. Start the environment only when runtime verification is needed. Read
   `references/dev-environment.md`, run the appropriate Bazel lifecycle target,
   then load and report the generated `deploy/dev/.env` values.

## Result

Report the absolute worktree path, branch, base ref and commit, whether the
development stack was started, and the generated ports when applicable. Do not
remove the worktree automatically; cleanup must preserve uncommitted or
unpushed work.
