# RFC-004: 终端输出带宽优化 - VirtualTerminal Serialize 模式

| 属性 | 值 |
|------|-----|
| **状态** | Proposed |
| **作者** | AgentsMesh Team |
| **创建日期** | 2026-01-24 |
| **目标** | 通过 VT Serialize 模式减少 30-50% 终端输出带宽 |

---

## 1. 概述

### 1.1 背景

AgentsMesh Runner 通过 PTY 捕获 AI Agent（如 Claude Code）的终端输出，并通过 gRPC/Relay 传输到前端浏览器。在实际运行中，发现终端输出流量非常大：

**实测数据（Claude Code 正常使用场景）：**
- 单帧大小：~880KB（包含完整终端界面）
- 帧频率：~0.85 帧/秒
- 带宽消耗：~800KB/s
- 7 分钟 session：~113MB 总流量

### 1.2 问题分析

Claude Code 使用 Synchronized Output 模式（`ESC[?2026h` ... `ESC[?2026l`）进行原子帧更新。每次更新都发送完整的终端界面，包含：

- Logo 和品牌区域（大量空格）
- 历史对话内容
- 状态栏和工具栏
- 代码编辑区域

**流量大的根本原因：**
1. 每帧包含完整屏幕内容（非差分）
2. 终端界面有大量空白区域（空格字符）
3. 当前使用原始 PTY 数据直接传输

### 1.3 已实现的优化

在 RFC 编写前，已实现 **带宽感知滑动窗口节流**（MR !46）：

- 检测高频全屏重绘模式
- 根据带宽动态调整节流窗口（1-4 秒）
- 跳过中间帧，只发送最新状态
- 预期减少 70-90% 流量（高频重绘场景）

**节流的局限性：**
- 对正常使用场景（非高频重绘）效果有限
- 每帧本身的大小（~880KB）未减少
- 只是减少发送频率，不减少单帧体积

### 1.4 目标

通过启用 VirtualTerminal Serialize 模式，在不丢失数据的前提下：

- 减少单帧体积 **30-50%**
- 与现有节流机制叠加，总带宽减少 **50-80%**
- 无需前端改动（输出仍是标准 ANSI 序列）

---

## 2. 技术方案

### 2.1 现有架构

```
PTY 输出 → OutputHandler → VirtualTerminal.Feed() [状态追踪]
                        → SmartAggregator.Write(rawData) → Relay/gRPC → Browser
```

当前 SmartAggregator 在 Legacy 模式下直接转发原始 PTY 数据。

### 2.2 Serialize 模式架构

```
PTY 输出 → OutputHandler → VirtualTerminal.Feed() [状态追踪]
                        → SmartAggregator.Write(nil) [标记有数据]
                                    ↓
                        → VirtualTerminal.Serialize() [压缩空格]
                                    ↓
                        → Relay/gRPC → Browser
```

### 2.3 空格压缩原理

VirtualTerminal.Serialize() 使用 CSI CUF（Cursor Forward）序列替代连续空格：

| 原始数据 | Serialize 输出 | 压缩比 |
|---------|---------------|--------|
| 100 个空格 | `\x1b[100C` | 100:6 = **94%** |
| 50 个空格 | `\x1b[50C` | 50:5 = **90%** |
| 10 个空格 | `\x1b[10C` | 10:5 = **50%** |

**代码实现（已存在）：**

```go
// runner/internal/terminal/serialize.go:113-201
func (h *StringSerializeHandler) nextCell(cell, oldCell Cell, row, col int) {
    // ...
    if isEmptyCell {
        h.nullCellCount += int(width)
    } else {
        if h.nullCellCount > 0 {
            // 使用 CUF 代替空格
            h.currentRow.WriteString(fmt.Sprintf("\x1b[%dC", h.nullCellCount))
            h.nullCellCount = 0
        }
        // 输出实际字符
        h.currentRow.WriteRune(cell.Char)
    }
}
```

### 2.4 启用方式

只需在 `message_handler.go` 中添加 `WithSerializeCallback`：

```go
// runner/internal/runner/message_handler.go

aggregator := terminal.NewSmartAggregator(grpcHandler, nil,
    terminal.WithFullRedrawThrottling(),
    terminal.WithSerializeCallback(func() []byte {
        return []byte(vt.Serialize(terminal.SerializeOptions{
            ScrollbackLines:  0,     // 不包含历史
            ExcludeAltBuffer: false, // 包含 alt screen
            ExcludeModes:     true,  // 不包含模式序列
        }))
    }),
)
```

### 2.5 Serialize 模式工作流程

1. **Write()**: 只标记 `hasPendingData = true`，不缓存原始数据
2. **flushLocked()**: 调用 `serializeCallback()` 获取压缩数据
3. **VT.Serialize()**: 遍历 Cell 缓冲区，将空格转换为 CUF 序列
4. **输出**: 标准 ANSI 转义序列，xterm.js 可直接渲染

---

## 3. 预期效果

### 3.1 压缩效果估算

Claude Code 界面分析（80x24 终端）：

| 区域 | 大小（字符） | 空格占比 | 压缩后 |
|------|------------|---------|--------|
| Logo/品牌 | ~300 | 70% | ~30% |
| 状态栏 | ~160 | 40% | ~70% |
| 代码区域 | ~1000 | 30% | ~80% |
| 对话历史 | ~500 | 50% | ~60% |

**综合压缩比：约 40-60%**

### 3.2 带宽减少对比

| 场景 | 原始 | 节流后 | Serialize 后 | 节流+Serialize |
|------|------|--------|-------------|----------------|
| 正常使用 | 800KB/s | 800KB/s | 400KB/s | 400KB/s |
| 高频重绘 | 2MB/s | 400KB/s | 1MB/s | 200KB/s |

### 3.3 不影响的场景

- 连接/重连时的 Snapshot 发送（已使用 Serialize）
- 前端渲染（标准 ANSI 序列）
- 增量更新帧（空格较少时效果有限）

---

## 4. 实现计划

### 4.1 Phase 1: 启用 Serialize 模式（低风险）

**变更：**
```go
// message_handler.go
aggregator := terminal.NewSmartAggregator(grpcHandler, nil,
    terminal.WithFullRedrawThrottling(),
    terminal.WithSerializeCallback(func() []byte {
        return []byte(vt.Serialize(terminal.SerializeOptions{
            ScrollbackLines:  0,
            ExcludeAltBuffer: false,
            ExcludeModes:     true,
        }))
    }),
)
```

**测试：**
- 单元测试：已有 `smart_aggregator_serialize_test.go`
- 集成测试：PTY 日志对比压缩前后流量
- 手工测试：Claude Code 正常使用场景

### 4.2 Phase 2: 监控与调优（可选）

- 添加 Prometheus metrics：`runner_terminal_output_bytes_total`
- 添加压缩比监控：`runner_serialize_compression_ratio`
- 根据实际数据调整 ScrollbackLines 参数

### 4.3 Phase 3: 差分编码（未来，高复杂度）

如果需要进一步优化，可考虑差分编码：
- 只发送与上一帧的差异
- 需要前端配合重建完整帧
- 预期额外减少 50-70%

---

## 5. 风险评估

| 风险 | 可能性 | 影响 | 缓解措施 |
|------|--------|------|----------|
| 压缩后渲染异常 | 低 | 中 | 单元测试覆盖、灰度发布 |
| 性能开销增加 | 低 | 低 | Serialize 已优化，开销可忽略 |
| 与节流机制冲突 | 无 | - | Serialize 模式下节流被绕过（设计如此） |

**关键点：** Serialize 模式在现有代码中已实现并测试，只需启用。

---

## 6. 替代方案对比

| 方案 | 复杂度 | 风险 | 效果 | 建议 |
|------|--------|------|------|------|
| **VT Serialize** | 低 | 低 | 30-50% | ✅ 推荐 |
| 更激进节流 | 低 | 低 | 有限 | 已���现 |
| 差分编码 | 高 | 高 | 70-90% | 未来考虑 |
| 帧大小限制 | 低 | 高 | 不确定 | 不推荐 |

---

## 7. 结论

VirtualTerminal Serialize 模式是当前最佳的带宽优化方案：

1. **已实现** - 代码已存在，只需启用
2. **低风险** - 无架构改动，输出仍是标准 ANSI
3. **效果明显** - 预期 30-50% 带宽减少
4. **可叠加** - 与现有节流机制互补

建议作为下一步优化实施。

---

## 附录

### A. 相关代码文件

| 文件 | 说明 |
|------|------|
| `runner/internal/terminal/serialize.go` | Serialize 核心实现 |
| `runner/internal/terminal/smart_aggregator.go` | WithSerializeCallback 选项 |
| `runner/internal/terminal/virtual_terminal.go` | VT 状态管理 |
| `runner/internal/runner/message_handler.go` | 启用位置 |

### B. 相关 MR

- MR !46: feat(runner): add bandwidth-aware full redraw throttling for PTY output
