---
name: gh-merge
description: |
  将当前分支合并到 GitHub 目标分支（通常是 main）。
  自动处理代码提交、创建 PR、监控 CI Checks、处理错误直到合并成功。
user-invocable: true
---

# GitHub 合并代码流程

将当前分支的代码通过 Pull Request 合并到 GitHub 目标分支。

## 使用方式

```
/gh-merge                 # 合并到 main（默认）
/gh-merge develop         # 合并到 develop 分支
/gh-merge --squash        # 使用 squash 方式合并
```

## 使用流程

### 1. 确认状态

```bash
git status
git branch --show-current
git remote -v
gh auth status
```

**前置检查：**
- 当前分支不能是目标分支
- `gh` CLI 已认证
- **识别 GitHub remote 名称**：检查 `git remote -v` 输出，找到指向 `github.com` 的 remote（可能是 `origin`、`github` 或其他名称），后续所有 git push/fetch 命令都使用该 remote 名称

### 2. 提交代码

如有未提交的更改，先提交：

```bash
git add <files>
git commit -m "feat/fix/refactor: 描述更改内容"
git push -u <github-remote> <current-branch>
```

**注意：**
- 优先 `git add <具体文件>` 而非 `git add .`，避免意外提交敏感文件
- push 时使用步骤 1 识别的 GitHub remote 名称

### 3. Rebase 到最新目标分支

合并前先 rebase，减少冲突风险：

```bash
git fetch <github-remote> main

# 如有未提交的更改，先 stash
git stash  # 仅在有 unstaged changes 时执行

git rebase <github-remote>/main

# rebase 完成后恢复 stash
git stash pop  # 仅在之前执行了 stash 时

# rebase 改变了历史，需要 force push
git push --force-with-lease <github-remote> <current-branch>
```

**如果 rebase 有冲突：**
1. 解决冲突文件
2. `git add <resolved-files>`
3. `git rebase --continue`
4. 重复直到 rebase 完成
5. `git push --force-with-lease`

### 4. 创建 Pull Request

```bash
gh pr create --base main --title "PR标题" --body "描述"
```

记录返回的 PR 编号（如 `#42`）。

### 5. 等待并监控 CI Checks（关键步骤）

**⚠️ 绝对不能跳过此步骤。必须确认 CI 全部通过后才能合并。**

```bash
# 等待 CI 触发（GitHub Actions 有 10-30 秒延迟）
sleep 30

# 等待所有 checks 完成
gh pr checks <pr-number> --watch --interval 15 --fail-fast
```

**处理 `no checks reported` 的情况：**

如果 `gh pr checks` 返回 `no checks reported`，这表示 CI 尚未触发，**绝不能**认为"没有 CI"而直接合并。必须重试：

```bash
# 等待更长时间后重试（最多重试 3 次，每次间隔 30 秒）
sleep 30
gh pr checks <pr-number> --watch --interval 15 --fail-fast
```

如果重试 3 次（共等待约 2 分钟）后仍然 `no checks reported`，则明确告知用户"CI 未触发"，**询问用户是否确认合并**，不得自行决定。

### 6. 处理 CI 失败

如果 CI Checks 失败：

```bash
# 1. 查看失败的 job 日志
gh run view <run-id> --log-failed

# 2. 修复代码

# 3. 提交修复并推送
git add <files>
git commit -m "fix: 修复 CI 错误"
git push <github-remote> <current-branch>

# 4. 重新等待 CI
sleep 30
gh pr checks <pr-number> --watch --interval 15 --fail-fast
```

重复此过程直到所有 Checks 通过。

### 7. 合并 PR

**前置条件（全部满足才能执行合并）：**
1. `gh pr checks` 至少报告了 1 个 check
2. 所有 checks 状态为 `pass`
3. 没有未解决的冲突

```bash
gh pr merge <pr-number> --squash --delete-branch
```

**合并策略：**
- `--squash`：压缩为单个 commit（推荐，保持历史整洁）
- `--merge`：保留完整 commit 历史
- `--rebase`：线性历史，无 merge commit

**处理 worktree 环境下的报错：**

在 git worktree 中执行 `gh pr merge --delete-branch` 时，可能报错：
```
failed to run git: fatal: 'main' is already used by worktree at '...'
```
这是因为 worktree 无法切换到 main 分支来删除本地分支。**这个报错不影响远程合并**，PR 已经成功合并。用 `gh pr view` 确认：

```bash
gh pr view <pr-number> --json state,mergedAt
# 确认 state 为 "MERGED"
```

### 8. 清理

```bash
# 在 worktree 中无需手动切换分支和清理
# worktree 会在退出时提示清理

# 非 worktree 环境：
git checkout main && git pull
git branch -d <branch-name>
```

## 完成后输出

```
✅ PR #42 已成功合并到 main

合并详情:
- 分支: feature/xxx → main
- CI Checks: 全部通过 (N/N)
- 合并方式: squash
- PR: https://github.com/org/repo/pull/42
```

## 处理常见问题

### PR 有合并冲突

```bash
git fetch <github-remote> main
git stash  # 如有 unstaged changes
git rebase <github-remote>/main
# 解决冲突...
git add <resolved-files>
git rebase --continue
git stash pop  # 如之前 stash 了
git push --force-with-lease <github-remote> <current-branch>
```

### CI 需要 re-run

```bash
gh run rerun <run-id> --failed
sleep 30
gh pr checks <pr-number> --watch --interval 15 --fail-fast
```

### Review 未通过

```bash
gh pr view <pr-number> --comments
# 修复后推送
git add <files> && git commit -m "fix: address review feedback" && git push
```

## 常用命令速查

| 操作 | 命令 |
|------|------|
| 查看 PR 列表 | `gh pr list` |
| 查看 PR 详情 | `gh pr view <number>` |
| 查看 PR Checks | `gh pr checks <number>` |
| 查看 Run 日志 | `gh run view <run-id> --log-failed` |
| 合并 PR | `gh pr merge <number> --squash --delete-branch` |
| 关闭 PR | `gh pr close <number>` |
| 重跑失败 CI | `gh run rerun <run-id> --failed` |

## 注意事项

- **CI 通过是合并的硬性前提**，绝不能在 CI 未完成或未触发时合并
- 识别正确的 GitHub remote 名称（不一定是 `origin`）
- 推荐 `--squash` 合并方式保持 main 历史整洁
- 使用 `--force-with-lease`（而非 `--force`）推送 rebase 后的代码
- worktree 环境中 `gh pr merge` 的本地分支删除报错可以忽略，远程合并不受影响
- 提交前确保代码已通过本地测试
