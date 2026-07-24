---
name: github-gitlab-mirror
description: >-
  Audits or synchronizes the authoritative GitHub main branch to the internal
  GitLab mirror without importing GitLab-only history back into GitHub. Use
  when checking GitHub/GitLab consistency, updating the internal mirror, or
  resolving a divergence between the two remotes.
---

# GitHub to GitLab Mirror

GitHub `main` is the single source of truth. GitLab exists to build and deploy
the same commit internally.

## Safety Rules

- Discover both remotes from their URLs; do not assume they are named `origin`
  and `gitlab`.
- Never merge, cherry-pick, or push GitLab-only commits into GitHub.
- A fast-forward update to GitLab may proceed after verification.
- Rewriting a divergent GitLab branch requires explicit user authorization and
  an exact `--force-with-lease` expectation.
- Do not change the local branch or working tree to mirror remote refs.

## Status

1. Fetch both authoritative refs:

   ```bash
   git remote -v
   git fetch <github-remote> main
   git fetch <gitlab-remote> main
   ```

2. Record both SHAs and show commits unique to either side:

   ```bash
   git rev-parse <github-remote>/main
   git rev-parse <gitlab-remote>/main
   git log --oneline <gitlab-remote>/main..<github-remote>/main
   git log --oneline <github-remote>/main..<gitlab-remote>/main
   ```

3. Classify the relationship:

   - Equal: already mirrored.
   - GitLab is an ancestor of GitHub: safe fast-forward.
   - GitHub is an ancestor of GitLab: GitLab contains mirror-only commits.
   - Neither is an ancestor: the remotes diverged.

## Synchronize

For a verified fast-forward:

```bash
git push <gitlab-remote> <github-remote>/main:refs/heads/main
```

For a GitLab-ahead or divergent branch, stop and display the GitLab-only
commits. An explicit request to restore GitLab as the GitHub mirror authorizes
discarding those mirror-only commits. Use the previously fetched GitLab SHA as
the lease:

```bash
git push \
  --force-with-lease=refs/heads/main:<expected-gitlab-sha> \
  <gitlab-remote> \
  <github-remote>/main:refs/heads/main
```

If the lease fails, fetch and reassess instead of weakening the protection.

## Verify

Fetch both remotes again and require exact SHA equality from fresh refs:

```bash
git fetch <github-remote> main
git fetch <gitlab-remote> main
test "$(git rev-parse <github-remote>/main)" = \
  "$(git rev-parse <gitlab-remote>/main)"
```

If GitHub advanced during the synchronization, return to the Status phase and
mirror the new SHA. Do not claim success until one fresh fetch of both remotes
observes equality.

Report both final SHAs and whether the GitLab build/deploy pipeline was
triggered. Do not describe the mirror as synchronized until the SHAs match.
The scheduled mirror job in `.gitlab-ci.yml` is the automation reference.
