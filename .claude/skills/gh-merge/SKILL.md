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
# 检查当前分支和未提交的更改
git status
git branch --show-current

# 确认目标分支（默认 main）
# 确认 remote 指向 GitHub
git remote -v
```

**前置检查：**
- 当前分支不能是目标分支（不能在 main 上对 main 创建 PR）
- 确认 `gh` CLI 已认证：`gh auth status`

### 2. 提交代码

如有未提交的更改，先提交：

```bash
# 添加所有更改
git add .

# 提交（使用有意义的 commit message）
git commit -m "feat/fix/refactor: 描述更改内容"

# 推送到远程
git push -u origin <current-branch>
```

### 3. 创建 Pull Request

使用 `gh` 创建 PR：

```bash
# 创建 PR 到 main 分支（交互式填充标题和描述）
gh pr create --base main --fill

# 或指定标题和描述
gh pr create --base main --title "PR标题" --body "描述"
```

记录返回的 PR 编号（如 `#123`）。

### 4. 监控 CI Checks

创建 PR 后，监控 CI 执行状态：

```bash
# 查看 PR 的 check 状态
gh pr checks <pr-number>

# 或查看 PR 详情
gh pr view <pr-number>

# 实时等待 checks 完成（最长等待 10 分钟）
gh pr checks <pr-number> --watch --interval 15 --fail-fast
```

### 5. 处理 CI 失败

如果 CI Checks 失败：

```bash
# 1. 查看失败原因
gh pr checks <pr-number>

# 2. 查看失败的 run 详细日志
gh run view <run-id> --log-failed

# 3. 根据错误修复代码
# ... 修复代码 ...

# 4. 提交修复
git add .
git commit -m "fix: 修复 CI 错误"
git push

# 5. 重新检查
gh pr checks <pr-number> --watch --interval 15 --fail-fast
```

重复此过程直到所有 Checks 通过。

### 6. 合并 PR

CI 通过后，合并 PR：

```bash
# 合并（squash commits）
gh pr merge <pr-number> --squash --delete-branch

# 或普通合并
gh pr merge <pr-number> --merge --delete-branch

# 或 rebase 合并
gh pr merge <pr-number> --rebase --delete-branch
```

**合并策略选择：**
- `--squash`：多个 commit 压缩为一个，保持历史整洁（推荐）
- `--merge`：保留完整 commit 历史，创建 merge commit
- `--rebase`：变基合并，线性历史，无 merge commit

### 7. 清理

合并成功后，清理本地分支和 worktree：

```bash
# 切回主分支
git checkout main
git pull

# 删除本地分支（合并后 --delete-branch 已删除远程分支）
git branch -d <branch-name>

# 如果是 worktree，删除 worktree
git worktree remove ../AgentsMesh-Worktrees/<dir-name>
```

## 完整示例

用户说："把当前分支合并到 main"

执行：
```bash
# 1. 检查状态
git status
git branch --show-current
# 假设当前分支是 feature/user-auth

# 2. 提交并推送
git add .
git commit -m "feat: add user authentication"
git push -u origin feature/user-auth

# 3. 创建 PR
gh pr create --base main --fill
# 返回: #42

# 4. 监控 CI
gh pr checks 42 --watch --interval 15 --fail-fast
# 等待 checks 完成...

# 5. 如果失败，修复后重新推送
# git add . && git commit -m "fix: ..." && git push

# 6. CI 通过后合并
gh pr merge 42 --squash --delete-branch

# 7. 清理
git checkout main && git pull
git branch -d feature/user-auth
```

## 完成后输出

```
✅ PR #42 已成功合并到 main

合并详情:
- 分支: feature/user-auth → main
- CI Checks: passed
- 合并方式: squash

已清理:
- 远程分支: feature/user-auth (已删除)
- 本地分支: feature/user-auth (已删除)
```

## 处理常见问题

### PR 有冲突

```bash
# 1. 拉取目标分支最新代码
git fetch origin main

# 2. 在当前分支上 rebase
git rebase origin/main

# 3. 解决冲突后继续
git add .
git rebase --continue

# 4. 强制推送（因为 rebase 改变了历史）
git push --force-with-lease
```

### Review 未通过

```bash
# 查看 review 评论
gh pr view <pr-number> --comments

# 修复后推送，通知 reviewer
git add . && git commit -m "fix: address review feedback" && git push
```

### CI 需要 re-run

```bash
# 重新运行失败的 workflow
gh run rerun <run-id> --failed
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
| 查看 PR 评论 | `gh pr view <number> --comments` |
| 重跑失败 CI | `gh run rerun <run-id> --failed` |

## 注意事项

- 提交前确保代码已通过本地测试
- PR 标题应清晰描述更改内容
- CI 失败时仔细阅读错误日志，使用 `gh run view --log-failed` 定位问题
- 合并前确认没有冲突
- 推荐 `--squash` 合并方式保持 main 历史整洁
- 合并后及时清理分支和 worktree
- 使用 `--force-with-lease`（而非 `--force`）推送 rebase 后的代码，更安全
