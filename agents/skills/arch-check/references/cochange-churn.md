# Lens E —— 时间维度：共变更 / churn

静态切面看不到的架构问题，往往在**版本历史**里暴露。这是来自代码审查 git-history lens 的高信号视角。仅当处于 git 仓库时启用。

## 1. 共变更耦合 (Co-change / Logical Coupling)

**核心信号：总是一起改、却分属不同模块的文件 = 隐藏耦合。** 物理上分开，逻辑上绑死——说明概念边界划错了。

**探测：**
```
# 列出最近 N 次提交里，哪些文件经常出现在同一个 commit
git log --since='12 months ago' --name-only --pretty=format: \
  | grep -v '^$' | sort | uniq -c | sort -rn | head -40

# 更准：对一对可疑文件，看它们共同出现在多少 commit
git log --pretty=format:'%H' --name-only \
  | awk '...'   # 构建 file-pair 共现计数（或用 code-maat / git-of-theseus）
```
工具：`code-maat`（专门做 logical coupling）、`git-of-theseus`。

**判读：**
- 两文件共变更率高（如 >60% 的改动同时碰它俩）但分属不同模块/目录 → 报"隐藏耦合：概念边界与改动边界不一致"，建议合并或显式化接口。
- 证据：给出共现的若干 commit hash + 两文件路径。

## 2. 上帝文件 / 变更热点 (Churn Hotspot)

**核心信号：人人都动的文件 = 职责过载或抽象失败。**

**探测：**
```
# 按改动次数排序文件（churn）
git log --since='12 months ago' --name-only --pretty=format: \
  | grep -v '^$' | sort | uniq -c | sort -rn | head -20

# 结合体量：高 churn × 大文件 = 最该拆的热点
```
**判读：** churn 排名靠前 + 行数大 + 触碰它的作者多 → 上帝文件，优先级高于普通"文件过大"。

## 3. 不稳定抽象 (Unstable Abstraction)

**核心信号：本应稳定的底层/领域模型却高频变动 = 抽象没立住。**

**探测：** 对照「依赖方向」稳定性排序——越靠近领域模型/被依赖越多的模块，churn 应越低。
```
git log --since='12 months ago' --name-only --pretty=format: \
  | grep 'src/domain/' | sort | uniq -c | sort -rn
```
**判读：** 被很多模块依赖（in-degree 高）但又高频改动的模块 → "不稳定抽象"，每次它一改就涟漪扩散。报出来并建议固化其接口。

## 启用与降级

- **在 git 仓库** → 跑全部三项（#1 共变更尽量用 code-maat）。
- **不在 git 仓库 / 浅克隆无历史** → 跳过本 lens，并在报告里注明"未做时间维度分析"。

> 共变更/churn 的结论也要接地：给出真实 commit hash 与文件路径，别报"我觉得这俩经常一起改"。
