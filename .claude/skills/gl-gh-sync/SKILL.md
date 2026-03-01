---
name: gl-gh-sync
description: |
  在 GitLab（内部）和 GitHub（开源）之间双向同步代码。
  自动检测哪边有新提交，通过各平台标准流程（MR/PR）合入，确保两边 main 分支保持一致。
user-invocable: true
---

# GL-GH-Sync —— GitLab / GitHub 双仓库同步流程

将 GitLab（origin）和 GitHub（github）的 main 分支保持同步。
两边都走各自平台的标准合入流程（GitLab 用 MR，GitHub 用 PR）。

## 使用方式

```
/sync                    # 自动检测方向并同步
/sync --to-github        # 强制将 GitLab 的变更同步到 GitHub
/sync --to-gitlab        # 强制将 GitHub 的变更同步到 GitLab
/sync --status           # 仅查看两边差异，不执行同步
```

## 同步流程

### 第一步：获取最新状态

```bash
# 同时拉取两个远端最新状态
git fetch origin main
git fetch github main

# 查看两边差异
git log --oneline origin/main..github/main  # GitHub 有但 GitLab 没有的提交
git log --oneline github/main..origin/main  # GitLab 有但 GitHub 没有的提交
```

### 第二步：判断同步方向

分析差异，确定操作：

```bash
GITLAB_AHEAD=$(git rev-list --count github/main..origin/main)
GITHUB_AHEAD=$(git rev-list --count origin/main..github/main)

echo "GitLab 领先 GitHub: $GITLAB_AHEAD 个提交"
echo "GitHub 领先 GitLab: $GITHUB_AHEAD 个提交"
```

根据结果：
- **两边相同**（均为 0）→ 已同步，无需操作
- **仅 GitLab 领先**→ 同步方向：GitLab → GitHub
- **仅 GitHub 领先**→ 同步方向：GitHub → GitLab
- **两边都有新提交（分叉）**→ 需人工决策，询问用户

### 第三步-A：GitLab → GitHub 同步

GitLab 有新提交，需要同步到 GitHub：

```bash
BRANCH="sync/gitlab-to-github-$(date +%Y%m%d-%H%M%S)"

# 基于 GitLab main 创建同步分支
git checkout -b "$BRANCH" origin/main

# 推送到 GitHub
git push github "$BRANCH"

# 在 GitHub 创建 PR
gh pr create \
  --repo AgentsMesh/AgentsMesh \
  --base main \
  --head "$BRANCH" \
  --title "sync: GitLab → GitHub ($(date +%Y-%m-%d))" \
  --body "$(cat <<'EOF'
## 同步说明

将 GitLab 内部仓库的最新提交同步到 GitHub 开源版本。

### 包含的提交

$(git log --oneline github/main..origin/main)

---
*此 PR 由自动同步流程创建*
EOF
)"
```

记录 PR 编号，然后监控 CI：

```bash
# 查看 PR 状态和 CI
gh pr checks <PR编号> --watch

# CI 通过后合并 PR
gh pr merge <PR编号> --merge --delete-branch

# 拉回本地，确认 GitHub main 已更新
git fetch github main
git log --oneline github/main -3
```

### 第三步-B：GitHub → GitLab 同步

GitHub 有新提交，需要同步到 GitLab：

```bash
BRANCH="sync/github-to-gitlab-$(date +%Y%m%d-%H%M%S)"

# 基于 GitHub main 创建同步分支
git checkout -b "$BRANCH" github/main

# 推送到 GitLab
git push origin "$BRANCH"

# 在 GitLab 创建 MR
glab mr create \
  --source-branch "$BRANCH" \
  --target-branch main \
  --title "sync: GitHub → GitLab ($(date +%Y-%m-%d))" \
  --description "将 GitHub 开源仓库的最新提交同步到 GitLab 内部版本。

### 包含的提交

$(git log --oneline origin/main..github/main)

---
*此 MR 由自动同步流程创建*" \
  --yes
```

记录 MR 编号，然后监控 Pipeline：

```bash
# 查看 MR 状态
glab mr view <MR编号>

# 查看 Pipeline 状态
glab ci status

# Pipeline 通过后合并 MR
glab mr merge <MR编号> --merge-commit-message "sync: GitHub → GitLab"

# 拉回本地，确认 GitLab main 已更新
git fetch origin main
git log --oneline origin/main -3
```

### 第四步：验证同步结果

```bash
git fetch origin main
git fetch github main

DIFF=$(git rev-list origin/main...github/main --count)
if [ "$DIFF" -eq 0 ]; then
  echo "✅ 同步成功：两边 main 分支完全一致"
  git log --oneline -3 origin/main
else
  echo "⚠️ 仍有差异，请检查"
fi
```

## 处理分叉情况（两边都有新提交）

当 GitLab 和 GitHub 都有对方没有的提交时：

```bash
echo "⚠️ 两个仓库出现分叉！"
echo ""
echo "GitLab 独有提交："
git log --oneline github/main..origin/main
echo ""
echo "GitHub 独有提交："
git log --oneline origin/main..github/main
```

**询问用户选择策略：**

1. **GitLab 为准**（内部版本优先）：以 GitLab main 为基础，将 GitHub 的独有提交挑选合并进来后同步回 GitHub
2. **GitHub 为准**（开源版本优先）：以 GitHub main 为基础，将 GitLab 的独有提交挑选合并进来后同步回 GitLab
3. **手动处理**：输出差异信息，由人工解决冲突后再执行同步

处理分叉示例（以 GitLab 为准）：

```bash
BRANCH="sync/resolve-diverge-$(date +%Y%m%d-%H%M%S)"

# 基于 GitLab main 创建解决分叉分支
git checkout -b "$BRANCH" origin/main

# 将 GitHub 的独有提交 cherry-pick 进来
# （需列出 GitHub 独有提交的 commit hash，逐个 cherry-pick）
git cherry-pick <github-commit-1> <github-commit-2> ...

# 如有冲突，解决后继续
git cherry-pick --continue

# 后续步骤同 "GitLab → GitHub 同步" 流程
```

## 处理 Pipeline/CI 失败

```bash
# GitLab Pipeline 失败
glab ci view        # 查看详细失败日志
glab ci retry       # 重试失败的 job

# GitHub Actions 失败
gh run list --branch "$BRANCH" --limit 5
gh run view <run-id> --log-failed   # 查看失败日志
gh run rerun <run-id> --failed      # 仅重试失败的 job
```

## 清理同步分支

同步完成后清理临时分支：

```bash
# 删除本地分支
git branch -d "$BRANCH"

# GitHub 分支（PR 合并时通常自动删除，手动删除：）
git push github --delete "$BRANCH"

# GitLab 分支（MR 合并时通常自动删除，手动删除：）
git push origin --delete "$BRANCH"
```

## 完整执行示例

用户说："把两个仓库同步一下"

```bash
# 1. 获取两边最新状态
git fetch origin main && git fetch github main

# 2. 检查差异
GITLAB_AHEAD=$(git rev-list --count github/main..origin/main)
GITHUB_AHEAD=$(git rev-list --count origin/main..github/main)
echo "GitLab 领先: $GITLAB_AHEAD | GitHub 领先: $GITHUB_AHEAD"

# 3. GitLab 有新提交 → 同步到 GitHub
if [ "$GITLAB_AHEAD" -gt 0 ] && [ "$GITHUB_AHEAD" -eq 0 ]; then
  BRANCH="sync/gitlab-to-github-$(date +%Y%m%d-%H%M%S)"
  git checkout -b "$BRANCH" origin/main
  git push github "$BRANCH"
  gh pr create --repo AgentsMesh/AgentsMesh --base main --head "$BRANCH" \
    --title "sync: GitLab → GitHub ($(date +%Y-%m-%d))" \
    --body "同步 GitLab 最新提交到 GitHub"
  gh pr checks --watch
  gh pr merge --merge --delete-branch
  git branch -d "$BRANCH"

# 4. GitHub 有新提交 → 同步到 GitLab
elif [ "$GITHUB_AHEAD" -gt 0 ] && [ "$GITLAB_AHEAD" -eq 0 ]; then
  BRANCH="sync/github-to-gitlab-$(date +%Y%m%d-%H%M%S)"
  git checkout -b "$BRANCH" github/main
  git push origin "$BRANCH"
  glab mr create --source-branch "$BRANCH" --target-branch main \
    --title "sync: GitHub → GitLab ($(date +%Y-%m-%d))" --yes
  glab ci status --live
  glab mr merge --merge-commit-message "sync: GitHub → GitLab"
  git branch -d "$BRANCH"
fi

# 5. 验证
git fetch origin main && git fetch github main
echo "最终状态 - origin/main: $(git rev-parse origin/main)"
echo "最终状态 - github/main: $(git rev-parse github/main)"
[ "$(git rev-parse origin/main)" = "$(git rev-parse github/main)" ] && \
  echo "✅ 两边完全一致" || echo "⚠️ 仍有差异"
```

## 完成后输出

```
✅ 同步完成

同步方向: GitLab → GitHub
同步分支: sync/gitlab-to-github-20260301-143022
PR: #42 (已合并)
CI: passed

最终状态:
  origin/main (GitLab): 7d8ecd45
  github/main (GitHub): 7d8ecd45

✅ 两边 main 分支完全一致
```

## 注意事项

- 同步前确保本地 main 是干净状态（无未提交修改）
- 分叉情况（两边都有独有提交）需要人工决策，避免自动覆盖
- 建议每次内部合并 MR 后立即触发同步，减少分叉概率
- GitLab CI 和 GitHub Actions 必须全部通过才能合并
- 同步分支命名格式：`sync/<direction>-<timestamp>`，便于识别和清理
- 如果两边的 main 都有 protected branch 规则，确保操作账户有合并权限
