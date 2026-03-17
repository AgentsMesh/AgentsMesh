---
title: "用 AgentsMesh 造 AgentsMesh：一个人 52 天的 Harness Engineering 实践"
excerpt: "OpenAI 称之为 Harness Engineering。我用这套方法论，一个人 52 天、600 次提交、965,687 行代码吞吐，建造了一个 Harness Engineering 工具本身。代码库即上下文，工程环境决定 Agent 的输出上限。"
date: "2026-03-04"
author: "AgentsMesh Team"
category: "Insight"
readTime: 12
---

OpenAI 最近发表了一篇文章，描述他们如何用 AI Agent 在5个月内产出超过百万行代码。他们把这套工程实践叫做 **Harness Engineering**。

我在50多天前开始建造 **AgentsMesh**。52天，600次提交，965,687行代码吞吐，现存356,220行代码。一个人。

但更值得说的不是数字，而是这件事本身的结构：我用 Harness Engineering 的方式，建造了一个 Harness Engineering 的工具。

仓库完全开源，git history 公开。所有数字可以用 git log 验证。

## 工程环境决定 Agent 的输出上限

52天的实际工作让我确信，Agent 的产出质量不只取决于 Agent 本身，更取决于它工作的工程土壤。这些是代码库里真实沉淀的选择。

### 分层架构，让 Agent 知道在哪里改

代码库是严格的 **DDD** 分层：domain 层只有数据结构，service 层只有业务逻辑，handler 层只有 HTTP 格式转换。22 个域模块，边界清晰，每个模块的 interface.go 明确定义了对外契约。

当 Agent 需要添加新功能，它知道：数据结构放 domain，业务规则放 service，路由放 handler。边界模糊的代码库，Agent 把东西放在错误的地方；边界清晰的代码库，Agent 的代码天然融合。这不是理论上的架构整洁，是 Agent 生成代码时的导航地图。

### 目录结构即文档

跨端命名完全对齐。拿 Loop 举例：backend/internal/domain/loop/ 是数据结构，backend/internal/service/loop/ 是业务逻辑，web/src/components/loops/ 是前端组件。产品概念到代码路径是直接映射，不需要搜索，目录名就是地图。

backend 的 16 个 domain 模块（agentpod、channel、ticket、loop、runner…）和 service 层 1:1 镜像；web 的 components 按产品功能分区（pod、tickets、loops、mesh、workspace），和 backend domain 命名对齐。一个 Agent 拿到 Ticket 相关的任务，不需要探索整个代码库，看目录就知道该动哪里。

这个约定不是文档写出来的，是整个代码库在每次 Agent 提交时持续强化的。

### 技术债务会被 Agent 指数级放大

这是 52 天里最反直觉的发现之一。

当你在某个模块做了一个临时性的妥协——绕过了正常的 service 层直接查数据库，或者用了一个 hardcode 的魔数——Agent 会把这个模式学走。下一次它生成类似功能的代码，它会复用这个'先例'。不是偶发的，是系统性的复制。技术债务不再是孤立的，它开始扩散。

人工工程师遇到糟糕的代码，通常知道'这是个坑，绕开它'。Agent 不会做这个判断——它看到的是：这个代码库里有这样的模式，所以这是合法的做法。

这意味着仓库里代码质量的信号比人写代码时重要得多。好的工程实践是主体，Agent 就放大好的工程实践；临时性的妥协是主体，Agent 就放大技术债务。

实际的工程应对：中间多次停下来专门清理技术债务，不发新功能，只做重构。不是为了让代码'好看'，是为了维护仓库里工程信号的纯度。这是 Agent 协作开发特有的维护成本，也是和传统开发节奏最大的不同之一。

### 强类型作为编译期质量闸

Go + TypeScript + Proto。强类型把大量错误从运行时前移到编译时。

Agent 生成了签名不匹配的函数？编译失败。Agent 修改了 API 格式忘了更新类型定义？TypeScript 直接报错。Agent 改了 Runner 的消息格式但没同步更新 Backend？Proto 生成的代码不能编译。

这些错误在弱类型语言里会悄悄进入运行时。强类型把它们挡在提交之前。反馈环路越短，Agent 的迭代效率越高。

### 四层反馈闭环

Agent 需要快速知道自己做错了什么。一层不够，四层刚好。而且反馈环路越短越精准，Agent 的交付结果越好。

第一层：编译。Air 热重载，Go 代码修改 1 秒内重启；TypeScript 类型错误实时标注。语法和类型级别的错误在这层清除。

第二层：单测。700+ 个测试覆盖 domain 和 service 层。Agent 修改后 5 分钟内知道有没有引入回归，尤其是多租户隔离这类容易被 Agent 遗漏的边界条件。

第三层：e2e。端到端验证真实的功能路径。覆盖 Agent 在单测里验证不到的集成边界——多个模块的真实联动。

第四层：CI pipeline。每个 PR 自动跑全量测试、lint、type-check、多平台构建验证。合并前的最后一道安全网，由机器执行，不依赖 review 者的细心程度。

四层延迟依次增加，覆盖的错误类型依次扩大。Agent 改一行代码，第一层确认；Agent 做跨模块重构，第四层才能完整验证。

### 工作树原生支持并行

dev.sh 根据 Git worktree 名自动计算端口偏移，每个工作树分配独立的端口区间。多个 Agent 在不同工作树上同时工作，环境完全隔离，不需要手工管理端口冲突。

这是 Pod 隔离原语在开发环境层面的延伸——同样的逻辑，从 Agent 执行环境贯穿到 Agent 开发环境。

### 代码库就是 Agent 的上下文，不只是 prompt

把上面这些放在一起，会发现它们指向同一个结论：代码库本身就是 Agent 工作时最重要的上下文。分层架构告诉 Agent 该在哪里改；目录结构告诉 Agent 该找哪个文件；技术债务的清洁程度决定 Agent 学到的是好模式还是坏模式；测试密度决定 Agent 有多大胆量重构；强类型决定 Agent 的错误能在多早被发现。

这意味着你不需要在代码库之外额外构建上下文系统——不需要刻意去做 Context Engineering，不需要单独搭 RAG，不需要维护额外的 memory 文件。你需要做的，是让代码库本身成为一个高质量的上下文。**仓库即 context。**

**这也是为什么 Harness Engineering 的投入方向，和传统软件工程是同一件事**：写清晰的代码、保持好的架构、及时清理技术债务。区别只是目的不同——以前是为了让人类工程师更容易维护，现在同时也是为了让 AI Agent 能够可信地工作。

## 认知带宽是真实的工程约束

第5天左右，我撞上了一堵真实的墙，5万行的日均代码吞吐。

三个 worktree 同时开着，三个 Agent 在跑，我在它们之间切换做决策。加第四个，决策质量明显下降。不是感觉，是后来发现那段时间留下了几个糟糕的架构决定。

日均5万行的吞吐量不是工具限制，是人类认知带宽的自然上限。你能为大约三条并行工作流做出真正的架构决策，超过这个数量，质量就开始下滑。

突破它的方式只有一个：用委托换规模。不是给 Agent 更多任务，而是委托决策本身。让 Agent 协调 Agent，自己上移一个层级——从监督单个 Agent，变成监督那个监督 Agent 的系统。所以我们有了 **Autopilot** 模式。

这是 AgentsMesh 的核心设计意图。也是我用它建造它自己的过程中，才真正理解的东西。

## 试错成本坍缩，工程方法论需要更新

AgentsMesh 的 Relay 架构不是设计出来的。是被生产环境打出来的。

三个 Pod 同时运行把 Backend 打挂。我看着它崩，理解了崩溃的原因，重新建。加了 Relay 隔离终端流量。新问题出现，加了智能聚合，加了按需连接管理。最终的架构来自一次次真实故障，不是白板讨论。

旧的工程直觉是先设计再建造——充分推演边界情况，因为出错代价高。

当试错成本接近零，这个直觉变成了负担。

那次 Relay 故障从发现到修复不到两天。在传统团队，这是两周的架构讨论——而且讨论一定会遗漏什么。

**AI 改变的不是写代码的速度，是整个工程过程的成本结构。**当迭代足够便宜，实验驱动比设计驱动产出更好的架构，更快。架构正确的标准不是通过评审，而是扛过生产环境。

## 自举验证

AgentsMesh 的核心主张：AI Agent 可以在有结构的 Harness 下，协作完成复杂的工程任务。

我用 AgentsMesh 建造了 AgentsMesh。

这是对主张最直接的检验。如果 Harness Engineering 真的有效，这个工具应该能够胜任建造自身。

52天，965,687行代码吞吐，现存356,220行生产代码，600次提交，一个作者。

OpenAI 是一支团队，用了5个月。这不是比较——场景不同，规模不同。但有一件事是一样的：Harness 让原本不可能的产出变得可能。

Commit history 是证据。任何工程师都可以 clone 仓库，git log --numstat，数字不会因为谁来看而改变。

## 三个工程原语

52 天的实践和自举验证，最终收敛出三个工程原语。这不是预先设计的产品框架，是被真实的工程问题逼出来的。

**隔离（Isolation）**
每个 Agent 需要自己独立的工作空间。不是最佳实践，是硬性前提。没有隔离，并行工作在结构上就是不可能的。AgentsMesh 用 **Pod** 实现这一点：每个 Agent 运行在独立的 Git worktree 和沙箱里。冲突从'可能发生'变成'结构上不能发生'。而隔离同时也意味着内聚，在独立的 Pod 环境中，把 Agent 运行需要的全部上下文准备好：Repo、Skills、MCP and more。实际上构建 Pod 的过程就是为 Agent 执行准备环境的过程。

**分解（Decomposition）**
Agent 不擅长处理'帮我搞这个代码库'。擅长的是：你拥有这个范围，这是验收标准，这是完成定义。所有权不只是任务分配，它改变了 Agent 推理的方式。分解是任何 Agent 运行之前必须完成的工程工作。

AgentsMesh 对分解提供了两种抽象：**Ticket** 对应一次性的工作项——功能开发、Bug 修复、重构，有完整的看板状态流转和 MR 关联；**Loop** 对应周期性的自动化任务——每日测试、定时构建、代码质量扫描，用 Cron 表达式调度，每次运行留下独立的 LoopRun 记录。两种任务形式边界清晰：做一件事，用 Ticket；反复做同一件事，用 Loop。

**协调（Coordination）**
我们没有使用岗位抽象来组织 Agent 的协同。传统团队需要岗位角色，是因为每个人只精通几个专业方向——前端工程师不写后端，产品经理不写代码。但 Agent 不受这个约束：同一个 Agent 可以写代码、生成文档、做竞品分析、执行测试、审查 PR，甚至编排其他 Agent 的工作流。它的能力边界不是固定的，是通过上下文和工具配置出来的。所以 Agent 之间的协同不需要模拟人类的分工方式，它需要的是通信和权限。

**Channel** 解决的是集体层次：多个 Pod 在同一个协作空间里共享消息、决策和文档。这是 Supervisor Agent 和 Worker Agent 能够形成协作结构的基础——不是群聊，是带上下文历史的结构化通信层。

**Binding** 解决的是能力层次：两个 Pod 之间点对点的权限授权。**pod:read** 让一个 Agent 可以观察另一个 Agent 的终端输出；**pod:write** 让一个 Agent 可以直接控制另一个 Agent 的执行。Binding 是 Agent 协调 Agent 的机制——Supervisor 不是靠发消息来感知 Worker 的状态，而是靠直接看到它的终端。

OpenAI 把相应的东西叫做 Context Engineering、架构约束和熵管理。名字不同，解决的是同一个问题。

## 开源

Harness Engineering 是一门工程学科，而非某个产品功能。与其敝帚自珍，不如抛砖引玉。

我们选择开源 AgentsMesh。当我们构建的可能是一种有效的工程工具时，目标从来不是'拥有代码'，而是让更多人能够在此基础上，构建出更好的工程工具。与其把可能正确的工程实践锁在产品里，不如开源它，让社区验证、演化、超越它。

代码在 [GitHub](https://github.com/AgentsMesh/AgentsMesh)

你可以用它来：部署自己的 Runner，在本地隔离环境里运行 AI Agent；用 Ticket 和 Loop 管理 Agent 的工作流；通过 Channel 和 Binding 让多个 Agent 协作完成复杂任务。

如果你在实践 Harness Engineering 的过程中有自己的发现——欢迎到 [GitHub Discussions](https://github.com/AgentsMesh/AgentsMesh/discussions) 聊，或者直接提 [Issue](https://github.com/AgentsMesh/AgentsMesh/issues)。这个项目本身就是用 Agent 建造的，它应该继续被 Agent 和工程师一起演进。
