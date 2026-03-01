---
name: merge
description: |
  Merge the current branch into the target branch (usually main).
  Automatically handles code commit, PR creation, CI monitoring, and error resolution until merge succeeds.
user-invocable: true
---

# Merge Code Flow

Merge the current branch into the target branch via a Pull Request.

## Workflow

### 1. Check Status

```bash
git status
git branch --show-current

# Confirm target branch (default: main)
```

### 2. Commit Code

If there are uncommitted changes, commit them first:

```bash
git add .
git commit -m "feat/fix/refactor: describe the change"
git push -u origin <current-branch>
```

### 3. Create Pull Request

Use `gh` to create a PR:

```bash
gh pr create --base main --title "PR title" --body "Description" --fill

# Or use the simplified command (auto-fills info)
gh pr create --fill
```

Note the returned PR number (e.g., `#42`).

### 4. Monitor CI

After creating the PR, monitor CI status:

```bash
gh pr checks <pr-number> --watch

# Or view PR status
gh pr view <pr-number>
```

### 5. Handle CI Failures

If CI fails:

```bash
# 1. View failure details
gh run list --branch <current-branch> --limit 5
gh run view <run-id> --log-failed

# 2. Fix the code based on the error
# ... fix code ...

# 3. Commit the fix
git add .
git commit -m "fix: resolve CI error"
git push

# 4. Re-check CI
gh pr checks <pr-number> --watch
```

Repeat until CI passes.

### 6. Merge PR

Once CI passes, merge the PR:

```bash
# Squash merge (recommended)
gh pr merge <pr-number> --squash --delete-branch

# Or regular merge
gh pr merge <pr-number> --merge --delete-branch
```

### 7. Cleanup (Optional)

After a successful merge, clean up local branch and worktree:

```bash
# Switch back to main repo
cd /path/to/AgentsMesh

# Delete local branch
git branch -d <branch-name>

# If using a worktree, remove it
git worktree remove ../AgentsMesh-Worktrees/<dir-name>
```

## Full Example

User says: "Merge the current branch into main"

```bash
# 1. Check status
git status
git branch --show-current
# Assume current branch is feature/user-auth

# 2. Commit and push
git add .
git commit -m "feat: add user authentication"
git push -u origin feature/user-auth

# 3. Create PR
gh pr create --base main --fill
# Returns: #42

# 4. Monitor CI
gh pr checks 42 --watch
# Wait for CI to complete...

# 5. If failed, fix and re-push
# git add . && git commit -m "fix: ..." && git push

# 6. Merge when CI passes
gh pr merge 42 --squash --delete-branch

# 7. Cleanup
cd /path/to/AgentsMesh
git worktree remove ../AgentsMesh-Worktrees/feature-user-auth
```

## Completion Output

```
PR #42 has been successfully merged into main

Merge details:
- Branch: feature/user-auth -> main
- CI: passed
- Merge method: squash

Cleaned up:
- Remote branch: feature/user-auth (deleted)
- Worktree: ../AgentsMesh-Worktrees/feature-user-auth (removed)
```

## Quick Reference

| Action | Command |
|--------|---------|
| List PRs | `gh pr list` |
| View PR details | `gh pr view <number>` |
| Check CI status | `gh pr checks <number>` |
| View CI logs | `gh run view <run-id> --log-failed` |
| Merge PR | `gh pr merge <number>` |
| Close PR | `gh pr close <number>` |

## Notes

- Ensure code passes local tests before committing
- PR titles should clearly describe the changes
- Read CI error logs carefully when pipeline fails
- Confirm no conflicts before merging
- Use `--squash` to combine multiple commits into one
- Clean up branches and worktrees after merging
