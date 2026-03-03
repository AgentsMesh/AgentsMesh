---
name: gl-gh-sync
description: |
  在 GitLab（内部）和 GitHub（开源）之间双向同步代码。
  自动检测哪边有新提交，fast-forward 场景直接推送，分叉场景走 PR/MR 流程。
user-invocable: true
---

# GL-GH-Sync —— GitLab / GitHub 双仓库同步流程

将 GitLab（origin）和 GitHub（github）的 main 分支保持同步。

**核心策略：**
- **一方领先（fast-forward）**→ 直接 `git push`，零额外 commit
- **两边分叉**→ 走 PR/MR 流程，需人工决策

## 使用方式

```
/gl-gh-sync              # 自动检测方向并同步
/gl-gh-sync --status     # 仅查看两边差异，不执行同步
```

## 同步流程

### 第一步：获取最新状态

```bash
git fetch origin main
git fetch github main

# 查看两边差异
git log --oneline origin/main..github/main  # GitHub 有但 GitLab 没有
git log --oneline github/main..origin/main  # GitLab 有但 GitHub 没有
```

### 第二步：判断同步方向

```bash
GITLAB_AHEAD=$(git rev-list --count github/main..origin/main)
GITHUB_AHEAD=$(git rev-list --count origin/main..github/main)

echo "GitLab 领先 GitHub: $GITLAB_AHEAD 个提交"
echo "GitHub 领先 GitLab: $GITHUB_AHEAD 个提交"
```

根据结果选择策略：
- **两边相同**（均为 0）→ 已同步，无需操作
- **仅 GitLab 领先**→ 直接推送到 GitHub
- **仅 GitHub 领先**→ 直接推送到 GitLab
- **两边都有新提交（分叉）**→ 需人工决策，询问用户

### 第三步-A：GitLab → GitHub（直接推送）

一方严格领先时，直接推送即可保持 commit hash 完全一致，无额外 merge commit。

```bash
# 直接将 GitLab main 推送到 GitHub main（fast-forward）
git push github origin/main:main
```

### 第三步-B：GitHub → GitLab（直接推送）

```bash
# 直接将 GitHub main 推送到 GitLab main（fast-forward）
git push origin github/main:main
```

> **注意：** 如果 GitLab main 有 protected branch 规则阻止直接推送，
> 则回退到 MR 流程（见"分叉处理"章节），但需注意 MR 合并产生的
> merge commit 需要再同步回 GitHub。

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

1. **GitLab 为准**（内部版本优先）：以 GitLab main 为基础，cherry-pick GitHub 独有提交，推送到两边
2. **GitHub 为准**（开源版本优先）：以 GitHub main 为基础，cherry-pick GitLab 独有提交，推送到两边
3. **手动处理**：输出差异信息，由人工解决冲突后再执行同步

处理分叉示例（以 GitLab 为准）：

```bash
BRANCH="sync/resolve-diverge-$(date +%Y%m%d-%H%M%S)"

# 基于 GitLab main 创建解决分叉分支
git checkout -b "$BRANCH" origin/main

# 将 GitHub 的独有提交 cherry-pick 进来
git cherry-pick <github-commit-1> <github-commit-2> ...

# 如有冲突，解决后继续
git cherry-pick --continue

# 推送到 GitLab（通过 MR 或直接推送）
git push origin "$BRANCH"
glab mr create --source-branch "$BRANCH" --target-branch main \
  --title "sync: resolve divergence ($(date +%Y-%m-%d))" --yes

# MR 合并后，将统一后的 main 直接推送到 GitHub
git fetch origin main
git push github origin/main:main
```

## 完整执行示例

```bash
# 1. 获取两边最新状态
git fetch origin main && git fetch github main

# 2. 检查差异
GITLAB_AHEAD=$(git rev-list --count github/main..origin/main)
GITHUB_AHEAD=$(git rev-list --count origin/main..github/main)
echo "GitLab 领先: $GITLAB_AHEAD | GitHub 领先: $GITHUB_AHEAD"

# 3a. GitLab 领先 → 直接推送到 GitHub
if [ "$GITLAB_AHEAD" -gt 0 ] && [ "$GITHUB_AHEAD" -eq 0 ]; then
  git push github origin/main:main

# 3b. GitHub 领先 → 直接推送到 GitLab
elif [ "$GITHUB_AHEAD" -gt 0 ] && [ "$GITLAB_AHEAD" -eq 0 ]; then
  git push origin github/main:main

# 3c. 两边分叉 → 需要人工决策
elif [ "$GITLAB_AHEAD" -gt 0 ] && [ "$GITHUB_AHEAD" -gt 0 ]; then
  echo "⚠️ 分叉！请选择同步策略"
fi

# 4. 验证
git fetch origin main && git fetch github main
[ "$(git rev-parse origin/main)" = "$(git rev-parse github/main)" ] && \
  echo "✅ 两边完全一致" || echo "⚠️ 仍有差异"
```

## 完成后输出

```
✅ 同步完成

同步方向: GitLab → GitHub
同步方式: 直接推送（fast-forward）

最终状态:
  origin/main (GitLab): ff6fef1d
  github/main (GitHub): ff6fef1d

✅ 两边 main 分支完全一致
```

## 注意事项

- 同步前确保本地 main 是干净状态（无未提交修改）
- 一方领先时直接推送，避免创建 PR/MR 引入额外 merge commit
- 分叉情况（两边都有独有提交）需要人工决策，避免自动覆盖
- 建议每次内部合并 MR 后立即触发同步，减少分叉概率
- 如果 protected branch 阻止直接推送，需回退到 PR/MR 流程
