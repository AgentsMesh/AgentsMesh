---
name: gh-merge
description: >-
  Completes the AgentsMesh GitHub pull-request workflow: commits scoped
  changes, rebases on the authoritative GitHub branch, opens or reuses a PR,
  monitors required checks, fixes failures, and merges only after verification.
  Use when the user asks to merge, submit, or land repository changes.
---

# GitHub Merge

GitHub is the development source of truth for this repository. Land changes
through a GitHub pull request; do not create a GitLab merge request for product
code.

## Workflow

1. Inspect repository state:

   ```bash
   git status --short --branch
   git branch --show-current
   git remote -v
   gh auth status --hostname github.com
   ```

2. Identify the GitHub remote by URL instead of assuming a remote name. Resolve
   the target branch, normally `main`, and refuse to merge directly while
   checked out on that branch.
3. Review all dirty files. Stage only files belonging to the requested change;
   preserve unrelated user changes and never use `git add .`. Run the relevant
   local tests and `git diff --check`, then create a focused commit containing
   only the requested change.
4. Require a clean index and working tree before rebasing. If unrelated user
   changes remain, do not stash or move them automatically; use an already
   isolated checkout or stop and report what prevents a safe rebase. Once the
   checkout is clean, fetch and rebase on the current GitHub target:

   ```bash
   git fetch <github-remote> <target-branch>
   git rebase <github-remote>/<target-branch>
   ```

   Resolve conflicts with user-owned work preserved, then rerun affected tests
   when the rebase changes the integration base.
5. Push the current branch. After a rebase, use `--force-with-lease`, never
   `--force`.
6. Reuse an existing pull request for the branch or create one with `gh pr
   create` and the repository PR template. The title and body must explain
   behavior, verification, and any remaining risk.
7. Inspect review state and wait for checks:

   ```bash
   gh pr checks <pr-number> --watch --interval 15 --fail-fast
   ```

   At least one check must be observed. If no checks are reported, retry after
   the workflow trigger delay; no checks is not a passing state. Diagnose
   failures from run logs, fix them, push, and monitor again.
8. Merge only when all observed checks have completed successfully, reviews and
   branch policy allow it, and the PR is conflict-free:

   ```bash
   gh pr merge <pr-number> --squash --delete-branch
   ```

9. Verify the remote PR state is `MERGED`. In a worktree, failure to delete the
   local branch does not mean the remote merge failed; report cleanup
   separately.

## Result

Report the PR URL, source and target branches, commit, local tests, remote check
results, merge method, and any local branch or worktree that remains.
