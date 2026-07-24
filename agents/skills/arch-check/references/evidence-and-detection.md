# 接地与探测手段（反幻觉）

LLM 架构审查的头号失败模式：**凭空断言结构**——声称"A 依赖 B""这里有循环""X 被多处写"，却没真去查。本文件给出强制接地原则 + 各语言的实际探测命令。

## 接地原则（不可豁免）

任何**结构性断言**都必须附带从真实代码里取到的证据，否则该 finding 置信度封顶 25 分（即不报）。结构性断言包括但不限于：

| 断言 | 必须出示的证据 |
|---|---|
| "A 依赖 B / 跨层依赖" | A 中 import/require/use B 的**那一行** `file:line` |
| "存在循环依赖" | 环上每条边的 import 行，或工具输出的环路径 |
| "X 是多写者 / 破坏 single-writer" | **每一处**写 X 的 `file:line` |
| "概念散落多处" | 该概念出现的**每个**位置 `file:line` |
| "derived 缓存无 invalidation" | 缓存写入点 + 基础属性写入点，证明后者不触发前者 |
| "上帝类 / 文件过大" | 实际行数（`wc -l`）/ 方法数 |
| "职责扩散 / 窃取" | 数据所在处 与 行为所在处 的两个 `file:line` |

> 一句话原则：**先搜索，再下结论。没有 `file:line` 的结构性判断不写进报告。**

下列命令以 POSIX shell 为示例。在原生 Windows 上，使用 `rg`、
PowerShell 的 `Get-ChildItem` / `Select-String` 或语言专用工具完成等价
探测，并保留相同的 `file:line` 证据；不得因示例命令不可用而跳过检查。

## 依赖 / 循环依赖探测

先用语言专用工具；没有就退回 grep import + 人工构图。

```
JS/TS     npx madge --circular src/        # 直接列出环
          npx madge --image graph.svg src/ # 依赖图
          (退回) grep -rn "^import\|require(" src/

Python    pydeps pkg --show-cycles
          (退回) grep -rn "^\(from\|import\) " pkg/

Go        go list -deps ./...   ；  go mod graph
          (环) 看 import 块 + go vet

Rust      cargo modules dependencies --lib
          (退回) grep -rn "^\(use\|mod\) " src/

Java/Kt   jdeps -verbose:class build/...   ；  深度可用 ArchUnit 写断言

通用      grep -rn 导入语句构边 → 手工找环；务必把证据行写进 finding
```

## single-writer / 多写者探测

对每个可疑可变状态 `X`：
```
grep -rn "X\s*=\|X\.set\|X\.store\|\.X\s*=\|mutate.*X\|push.*X" <scope>
```
- 写入点 = 1 → single-writer ✅
- 写入点 ≥ 2 且无锁/通道/actor 串行化 → 报多写者，**列全部写入点**。
- 别忘了间接写：通过 `&mut` 借出、通过 setter、通过共享引用别名。

## 概念散落 / co-located 检查

对一个领域概念名（如 `Order`/`Invoice`）：
```
grep -rln "Order" src/ | sort        # 出现在哪些文件
```
若该概念的**数据定义、规则、状态转换**分散在 ≥3 个互不内聚的位置，报"概念碎片化"，列出全部位置。

## 文件 / 模块体量

```
find <scope> -name '*.ts' | xargs wc -l | sort -rn | head   # 找最大文件
```
用真实数字对照 references 里的阈值（>200 理想线 / >400 拆分线 / 模块 >20 文件）。

## 报告里的证据格式

每条 finding 末尾保留一行 `证据：` 给出关键 `file:line`（或工具输出片段），让人能一键核对。审查者自己也用它做自检：写不出证据行 = 这条是幻觉，删掉。
