# RFC-005: AutopilotController 有监督自动化架构设计

| 属性 | 值 |
|------|-----|
| **状态** | Draft |
| **作者** | AgentsMesh Team |
| **创建日期** | 2026-01-28 |
| **目标** | 实现有监督的 AI Agent 自动化任务执行 |

---

## 1. 概述

### 1.1 背景

Claude Code 等 AI Agent 在交互模式下工作时，会在关键决策点暂停等待用户输入（如 Plan 审批、权限请求、异常处理等）。用户需要持续关注并手动响应这些请求，无法实现真正的自动化。

AutopilotController 旨在解决这一问题，通过引入一个"管理者"角色（Control Process）来监督和驱动"执行者"（Pod）完成任务，实现有监督的自动化执行。

### 1.2 目标

- 实现 AI Agent 的有监督自动化执行（Supervised Automation）
- Control Process 能够观察 Pod 状态并做出决策
- 支持任务完成检测、异常处理、熔断保护
- 保持 Control Process 的上下文连续性（通过 session 恢复）

### 1.3 非目标

- 完全无人值守的 fire-and-forget 模式
- 多 Pod 协作（单个 AutopilotController 对应单个 Pod）
- Control Process 的分布式部署（当前限定同机运行）

---

## 2. 产品定位

**AutopilotController** = 有监督的自动化执行控制器（Supervised Automation Controller）

| 角色 | 职责 |
|------|------|
| **Control Process** | 决策者/管理者，观察 Pod 状态，判断进展，发送指令 |
| **Pod** | 执行者，运行 Claude Code（交互模式）完成实际任务 |

**核心场景**：
- Pod 运行 Claude Code（交互模式或 Plan Mode）
- Pod 在关键点"卡住"等待决策（Plan 审批、权限请求、异常处理等）
- Control Process 观察 Pod 状态并做出决策
- Control Process 发送指令驱动 Pod 继续工作

---

## 3. 整体架构

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              Runner                                      │
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                         AutopilotController                                 │    │
│  │                                                                  │    │
│  │   ┌─────────────┐      ┌──────────────────┐                     │    │
│  │   │ Go 控制逻辑  │      │  Control Process │                     │    │
│  │   │             │      │  (Claude Code)   │                     │    │
│  │   │ - 生命周期   │      │                  │                     │    │
│  │   │ - session   │◄────►│  - 观察 Pod   │                     │    │
│  │   │ - 迭代计数   │      │  - 判断进展      │                     │    │
│  │   │ - 事件上报  │      │  - 发送指令      │                     │    │
│  │   └──────┬──────┘      └────────┬─────────┘                     │    │
│  │          │                      │                                │    │
│  │          │ StateChangeCallback  │ MCP HTTP (curl)               │    │
│  │          │                      │                                │    │
│  │   ┌──────▼──────────────────────▼─────────┐                     │    │
│  │   │           Pod                   │                     │    │
│  │   │                                        │                     │    │
│  │   │   ┌──────────────────────────────┐    │                     │    │
│  │   │   │    PTY Terminal              │    │                     │    │
│  │   │   │    ┌────────────────────┐    │    │                     │    │
│  │   │   │    │   Claude Code      │    │    │                     │    │
│  │   │   │    │   (交互模式)        │    │    │                     │    │
│  │   │   │    └────────────────────┘    │    │                     │    │
│  │   │   └──────────────────────────────┘    │                     │    │
│  │   │                                        │                     │    │
│  │   │   ┌──────────────────────────────┐    │                     │    │
│  │   │   │  TerminalStateDetector       │    │                     │    │
│  │   │   │  - 屏幕内容 hash 检测         │    │                     │    │
│  │   │   │  - 输入提示符识别             │    │                     │    │
│  │   │   │  - StateWaiting 回调          │    │                     │    │
│  │   │   └──────────────────────────────┘    │                     │    │
│  │   └────────────────────────────────────────┘                     │    │
│  └──────────────────────────────────────────────────────────────────┘    │
│                                                                          │
│  ┌────────────────────────────────────────┐                              │
│  │         MCP HTTP Server                │                              │
│  │  - get_pod_snapshot                  │                              │
│  │  - send_pod_input                    │                              │
│  │  - get_pod_status                      │                              │
│  └────────────────────────────────────────┘                              │
└─────────────────────────────────────────────────────────────────────────┘
```

### 3.1 组件说明

| 组件 | 说明 |
|------|------|
| **Go 控制逻辑** | AutopilotController 的核心，管理生命周期、session、事件上报 |
| **Control Process** | 完整的 Claude Code 实例（-p 模式），运行在 Pod 工作目录，拥有完整能力（读写文件、执行命令等），但角色定位是管理者 |
| **Pod** | 普通的 AgentPod，运行 Claude Code 执行实际任务 |
| **TerminalStateDetector** | 检测 Pod 终端状态，触发 Control 唤醒 |
| **MCP HTTP Server** | 提供终端观察和控制的 API |

---

## 4. 核心流程

### 4.1 Pod 与 AutopilotController 的关系

AutopilotController 是一个"附加控制器"，绑定到已存在的 Pod 上：

```
1. Pod 独立创建并运行
   - 用户创建 Pod
   - Claude Code 启动，进入等待状态
   │
   ▼
2. 用户决定让 Autopilot 控制该 Pod
   │
   ▼
3. 创建 AutopilotController，绑定到已有的 Pod
   - 设置 TerminalStateDetector 回调
   │
   ▼
4. 立即启动 Control Process（第一次）
   - claude -p "初始 prompt" --output-format json
   - Control 观察 Pod 状态
   - Control 发送第一个指令（或判断已完成）
   - Control 退出
   - 保存 session_id
   │
   ▼
5. 等待 Pod 执行
   - TerminalStateDetector 持续检测屏幕状态
   │
   ▼
6. TerminalStateDetector 检测到 StateWaiting
   │
   ▼
7. 触发 StateChangeCallback
   │
   ▼
8. Resume Control Process
   - claude --resume $session_id -p "Pod 已就绪"
   - Control 基于上下文观察、判断、决策
   - Control 退出
   │
   ▼
9. 循环 5-8，直到：
   - Control 判断任务完成 → PhaseCompleted
   - 达到最大迭代数 → PhaseMaxIterations
   - Control 决策 NEED_HUMAN_HELP → PhaseWaitingApproval
   - Control 决策 GIVE_UP → PhaseFailed
   - 用户手动停止 → PhaseStopped
```

### 4.2 Control Process 单次执行流程

```
Control Process 启动
   │
   ▼
1. 观察 Pod 终端
   - 调用 get_pod_snapshot 获取屏幕内容
   │
   ▼
2. 分析当前状态
   - Pod 在做什么？
   - 任务进展如何？
   - 是否有错误？
   │
   ▼
3. 做出决策
   │
   ├─► 任务完成 → 输出 TASK_COMPLETED → 退出
   │
   ├─► 需要继续 → 发送指令 → 输出 CONTINUE → 退出
   │
   ├─► 需要人工帮助 → 输出 NEED_HUMAN_HELP → 退出
   │
   └─► 无法继续 → 输出 GIVE_UP → 退出
```

### 4.3 状态机

```
                    ┌─────────────────┐
                    │  initializing   │
                    └────────┬────────┘
                             │ 初始化完成
                             ▼
                    ┌─────────────────┐
         ┌─────────│    running      │◄────────┐
         │         └────────┬────────┘         │
         │                  │                  │
    用户暂停           各种终止条件         用户恢复
         │                  │                  │
         ▼                  ▼                  │
┌─────────────────┐  ┌─────────────────┐       │
│     paused      │  │ waiting_approval│───────┘
└─────────────────┘  └─────────────────┘
         │                  │
         │                  │ 超时/拒绝
         │                  ▼
         │           ┌─────────────────┐
         │           │     failed      │
         │           └─────────────────┘
         │
         │ 用户停止         ┌─────────────────┐
         └────────────────►│    stopped      │
                           └─────────────────┘

                    ┌─────────────────┐
                    │   completed     │
                    └─────────────────┘

                    ┌─────────────────┐
                    │ max_iterations  │
                    └─────────────────┘
```

---

## 5. 关键组件设计

### 5.1 AutopilotController 结构体

```go
type AutopilotController struct {
    key           string
    podKey  string
    config        *runnerv1.AutopilotConfig

    // Pod 控制
    podCtrl    PodController
    stateDetector *terminal.TerminalStateDetector

    // Control Process 管理
    sessionID     string              // Claude Code session ID
    mcpPort       int                 // MCP HTTP Server 端口

    // 状态
    status        Status
    mu            sync.RWMutex

    // 生命周期
    ctx    context.Context
    cancel context.CancelFunc

    // 事件上报
    reporter EventReporter

    log *slog.Logger
}
```

### 5.2 状态检测 - TerminalStateDetector

使用 `TerminalStateDetector` 替代基于进程的 `Monitor`，通过分析终端屏幕内容检测 Pod 状态：

```go
// 初始化时设置回调
func (rp *AutopilotController) setupStateDetector() {
    rp.stateDetector = terminal.NewTerminalStateDetector(
        rp.podCtrl.GetVirtualTerminal(),
        terminal.WithStateChangeCallback(func(newState, prevState terminal.AgentState) {
            if newState == terminal.StateWaiting && prevState == terminal.StateExecuting {
                rp.OnPodWaiting()
            }
        }),
    )

    // 启动周期性检测
    go rp.runStateDetection()
}

func (rp *AutopilotController) runStateDetection() {
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-rp.ctx.Done():
            return
        case <-ticker.C:
            rp.stateDetector.DetectState()
        }
    }
}
```

**检测策略**：
1. 检查终端是否在 alt screen 模式（Claude Code TUI）
2. 计算屏幕内容 hash，检测变化
3. 屏幕稳定后，检测输入提示符
4. 检测到提示符 → `StateWaiting` → 触发回调

### 5.3 Control Process - Session 管理

**第一次启动**：

```go
func (rp *AutopilotController) startControlProcess(ctx context.Context) (*ControlDecision, error) {
    prompt := rp.buildPrompt()

    args := []string{
        "--dangerously-skip-permissions",
        "-p", prompt,
        "--output-format", "json",
    }

    cmd := exec.CommandContext(ctx, "claude", args...)
    cmd.Dir = rp.podCtrl.GetWorkDir()
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }

    // 解析 JSON 输出，获取 session_id
    var result struct {
        SessionID string `json:"session_id"`
    }
    json.Unmarshal(output, &result)
    rp.sessionID = result.SessionID

    return rp.parseDecision(output)
}
```

**Resume 启动**：

```go
func (rp *AutopilotController) resumeControlProcess(ctx context.Context) (*ControlDecision, error) {
    prompt := rp.buildResumePrompt()

    args := []string{
        "--dangerously-skip-permissions",
        "--resume", rp.sessionID,
        "-p", prompt,
        "--output-format", "json",
    }

    cmd := exec.CommandContext(ctx, "claude", args...)
    cmd.Dir = rp.podCtrl.GetWorkDir()
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }

    return rp.parseDecision(output)
}
```

### 5.4 Prompt 设计

**初始 Prompt（第一次启动）**：

```
你是任务编排代理。你的职责是监督一个 Pod（另一个运行在终端中的 Claude Code 实例）完成任务。

## 你的角色
- 你是管理者/决策者，不是执行者
- 你通过观察 Pod 的终端输出来了解进展
- 你通过发送文本指令来驱动 Pod 工作
- 每次决策后你会退出，等待下次被唤醒

## 重要限制
- **你不能直接完成任务！** 你必须通过 Pod 来完成所有工作
- 禁止直接读写文件（使用 Read/Write/Edit 工具）
- 禁止直接执行 git 命令或其他系统命令
- 你唯一允许使用的 Bash 命令是下面提供的 curl 命令，用于与 Pod 交互
- 如果你发现自己想要直接执行任务，停下来，改为向 Pod 发送指令

## 任务
{prompt}

## 与 Pod 交互的方式
使用 Bash 工具执行以下 curl 命令：

1. 观察 Pod 终端（最近 N 行）：
   curl -s -X POST -H "Content-Type: application/json" \
     "http://127.0.0.1:{mcp_port}/mcp" \
     -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_pod_snapshot","arguments":{"pod_key":"{pod_key}","lines":100}}}'

2. 发送输入给 Pod（文本+回车键）：
   curl -s -X POST -H "Content-Type: application/json" \
     "http://127.0.0.1:{mcp_port}/mcp" \
     -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"send_pod_input","arguments":{"pod_key":"{pod_key}","text":"你的指令","keys":["enter"]}}}'

## 工作流程
1. 观察 Pod 终端，了解当前状态
2. 判断任务是否完成
3. 如果完成，输出 TASK_COMPLETED
4. 如果未完成，发送下一步指令给 Pod
5. 输出你的决策摘要

## 输出格式
必须以以下格式之一结束：

TASK_COMPLETED
{完成摘要}

或

CONTINUE
{发送的指令和原因}

或

NEED_HUMAN_HELP
{为什么需要人工介入，遇到了什么问题}

或

GIVE_UP
{为什么放弃，遇到了什么无法解决的问题}

## 开始
请先观察 Pod 终端状态，然后做出决策。
```

**Resume Prompt（后续唤醒）**：

```
Pod 已完成上一步操作，现在处于等待输入状态。

当前进度：第 {iteration} 次迭代 / 最多 {max_iterations} 次

请继续：
1. 观察 Pod 终端，查看上一步的执行结果
2. 判断任务是否完成
3. 做出下一步决策
```

### 5.5 决策驱动的终止机制

**设计原则**："是否有进展"、"是否应该停止"是需要理解上下文的判断，应该由 Control（Claude Code）来决策，而不是用简单规则。

**Control 的输出格式**：

```
TASK_COMPLETED
{完成摘要}

或

CONTINUE
{发送的指令和原因}

或

NEED_HUMAN_HELP
{为什么需要人工介入，遇到了什么问题}

或

GIVE_UP
{为什么放弃，遇到了什么无法解决的问题}
```

**Go 代码的保护机制**：

Go 代码只保留最基本的保护——**最大迭代数**，防止 token 无限消耗：

```go
func (rp *AutopilotController) OnPodWaiting() {
    // 检查最大迭代数（唯一的硬性保护）
    if rp.status.CurrentIteration >= rp.status.MaxIterations {
        rp.setPhase(PhaseMaxIterations)
        return
    }

    rp.status.CurrentIteration++

    // 运行 Control Process
    decision, err := rp.runControlProcess(rp.ctx)
    if err != nil {
        // Control 执行失败，记录错误但继续
        rp.log.Error("Control process failed", "error", err)
        return
    }

    // 根据 Control 的决策处理
    switch decision.Type {
    case DecisionCompleted:
        rp.setPhase(PhaseCompleted)
    case DecisionContinue:
        // 等待下一次 Pod waiting
    case DecisionNeedHumanHelp:
        rp.setPhase(PhaseWaitingApproval)
    case DecisionGiveUp:
        rp.setPhase(PhaseFailed)
    }
}
```

**为什么不用规则引擎做熔断？**

- "是否有进展"需要理解任务上下文，简单的文件变更计数不够准确
- Control 本身就是一个有判断能力的 AI，应该让它自己决定何时需要帮助
- 规则可能误判（有进展但没改文件，或改了文件但没实际进展）

---

## 6. 与现有代码的变更

### 6.1 需要修改的文件

| 文件 | 变更 |
|------|------|
| `runner/internal/autopilot/autopilot_controller.go` | 添加 sessionID、stateDetector，重构启动逻辑 |
| `runner/internal/autopilot/control_process.go` | 分离 startControlProcess 和 resumeControlProcess，移除 helper scripts |
| `runner/internal/runner/message_handler.go` | 用 TerminalStateDetector 替代 claudeMonitor |
| `runner/internal/runner/pod_controller.go` | 添加 GetVirtualTerminal() 方法 |

### 6.2 需要删除的代码

| 文件 | 删除内容 |
|------|---------|
| `runner/internal/autopilot/autopilot_controller.go` | `pollForNextIteration()` 方法 |
| `runner/internal/autopilot/control_process.go` | `generateHelperScripts()` 方法 |
| `runner/internal/autopilot/circuit_breaker.go` | 整个文件（熔断改为 Control 决策） |
| `runner/internal/runner/message_handler.go` | `claudeMonitor` 相关的 AutopilotController 订阅代码 |

### 6.3 PodController 接口扩展

```go
type PodController interface {
    SendInput(text string) error
    GetWorkDir() string
    GetPodKey() string
    GetAgentStatus() string
    GetVirtualTerminal() *terminal.VirtualTerminal  // 新增
}
```

---

## 7. 安全考虑

### 7.1 移除 helper scripts

当前的 helper scripts 存在 shell 注入风险：

```bash
# 当前实现
TEXT="$1"  # 如果 $1 包含特殊字符，可能导致注入
```

**解决方案**：在 prompt 中直接提供 curl 命令模板，Control 使用 Bash 工具执行时自行构造参数。JSON 参数由 Control（Claude Code）生成，无需 shell 变量替换。

### 7.2 MCP 调用安全

- MCP HTTP Server 只监听 `127.0.0.1`
- Control Process 在同一台机器上运行
- 不暴露到外部网络
- 通过 `X-Pod-Key` header 进行简单的身份验证

---

## 8. 测试策略

### 8.1 单元测试

- TerminalStateDetector 状态检测和回调触发
- Prompt 构建和决策解析

### 8.2 集成测试

- AutopilotController 完整生命周期
- Control Process session 恢复
- 状态转换正确性

### 8.3 E2E 测试

- 创建 AutopilotController，验证 Control 第一次启动
- 验证 Pod 执行后 Control resume
- 验证任务完成检测（TASK_COMPLETED）
- 验证 Control 决策 NEED_HUMAN_HELP 和 GIVE_UP 的处理

---

## 9. 待确认问题

### 9.1 Claude Code CLI 行为验证

1. **`--resume` + `-p` 组合是否可用？**
   - 需要验证 `claude --resume $session_id -p "prompt"` 是否正常工作

2. **session_id 的获取方式**
   - `--output-format json` 输出中是否包含 session_id？
   - 如不包含，需要其他方式获取或持久化 session

### 9.2 VirtualTerminal 共享

- PodController 需要暴露 VirtualTerminal
- 需确认 VirtualTerminal 可以安全地跨组件共享

---

## 10. 未来扩展

### 10.1 多 Pod 协作

当前设计是 1:1（一个 AutopilotController 对应一个 Pod）。未来可扩展为 1:N，支持 Control 协调多个 Pod。

### 10.2 Control 分布式部署

当前 Control 限定与 Pod 同机运行。未来可考虑 Control 远程运行，通过网络 MCP 调用。
