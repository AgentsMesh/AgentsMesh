---
name: merge
description: |
  将当前分支合并到目标分支（通常是 main）。
  自动处理代码提交、创建 MR、监控 Pipeline、处理错误直到合并成功。
user-invocable: true
---

# 合并代码流程

将当前分支的代码通过 Merge Request 合并到目标分支。

## 使用流程

### 1. 确认状态

```bash
# 检查当前分支和未提交的更改
git status
git branch --show-current

# 确认目标分支（默认 main）
```

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

### 3. 创建 Merge Request

使用 `glab` 创建 MR：

```bash
# 创建 MR 到 main 分支
glab mr create --target-branch main --title "MR标题" --description "描述" --fill

# 或者使用简化命令（自动填充信息）
glab mr create -f
```

记录返回的 MR 编号（如 `!123`）。

### 4. 监控 Pipeline

创建 MR 后，监控 Pipeline 执行状态：

```bash
# 查看 Pipeline 状态
glab ci status

# 或查看 MR 状态
glab mr view <mr-number>
```

### 5. 处理 Pipeline 失败

如果 Pipeline 失败：

```bash
# 1. 查看失败原因
glab ci status
glab ci view  # 查看详细日志

# 2. 根据错误修复代码
# ... 修复代码 ...

# 3. 提交修复
git add .
git commit -m "fix: 修复 CI 错误"
git push

# 4. 重新检查 Pipeline
glab ci status
```

重复此过程直到 Pipeline 通过。

### 6. 合并 MR

Pipeline 通过后，合并 MR：

```bash
# 合并（squash commits）
glab mr merge <mr-number> --squash

# 或直接合并
glab mr merge <mr-number>
```

### 7. 清理（可选）

合并成功后，清理本地分支和 worktree：

```bash
# 切回主仓库
cd /path/to/AgentMesh

# 删除远程分支（MR 合并时通常自动删除）
git push origin --delete <branch-name>

# 删除本地分支
git branch -d <branch-name>

# 如果是 worktree，删除 worktree
git worktree remove ../AgentMesh-Worktrees/<dir-name>
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

# 3. 创建 MR
glab mr create --target-branch main --fill
# 返回: !42

# 4. 监控 Pipeline
glab ci status --live
# 等待 Pipeline 完成...

# 5. 如果失败，修复后重新推送
# git add . && git commit -m "fix: ..." && git push

# 6. Pipeline 通过后合并
glab mr merge 42 --squash

# 7. 清理
cd /path/to/AgentMesh
git worktree remove ../AgentMesh-Worktrees/feature-user-auth
```

## 完成后输出

```
✅ MR !42 已成功合并到 main

合并详情:
- 分支: feature/user-auth → main
- Pipeline: passed
- 合并方式: squash

已清理:
- 远程分支: feature/user-auth (已删除)
- Worktree: ../AgentMesh-Worktrees/feature-user-auth (已删除)
```

## 常用命令速查

| 操作 | 命令 |
|------|------|
| 查看 MR 列表 | `glab mr list` |
| 查看 MR 详情 | `glab mr view <number>` |
| 查看 Pipeline | `glab ci status` |
| 查看 CI 日志 | `glab ci view` |
| 合并 MR | `glab mr merge <number>` |
| 关闭 MR | `glab mr close <number>` |

## 注意事项

- 提交前确保代码已通过本地测试
- MR 标题应清晰描述更改内容
- Pipeline 失败时仔细阅读错误日志
- 合并前确认没有冲突
- 使用 `--squash` 可将多个 commit 合并为一个
- 合并后及时清理分支和 worktree
