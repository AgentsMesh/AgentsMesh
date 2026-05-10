# Relay Client Stop() 超时问题

**日期**: 2026-02-09

## 问题描述

Runner 日志中频繁出现 `Timeout waiting for relay loops to exit` 警告，导致 `Stop()` 方法需要等待 5 秒超时才能完成。

### 症状

```
time=22:26:58.592 INFO  "Stopping relay client"
time=22:26:58.592 ERROR "Read error" error="...use of closed network connection"
time=22:26:58.592 INFO  "Read loop exited"
time=22:26:58.592 INFO  "Write loop exited"
time=22:27:03.593 WARN  "Timeout waiting for relay loops to exit"  ← 5秒后才完成
time=22:27:03.594 INFO  "Relay client stopped"
```

## 根本原因

### 竞态条件

`Stop()` 和 `reconnectLoop()` 之间存在竞态条件：

1. **WaitGroup 计数不一致**: 当 `Stop()` 被调用时，`reconnectLoop` 可能正在执行 `wg.Add(2)` 启动新的 readLoop/writeLoop
2. **defer 执行顺序问题**: `readLoop` 中的 `wg.Done()` 在 defer 中最后执行，在 `onClose` 回调之后
3. **缺乏统一的"已停止"状态检查**: `Stop()` 设置 `stopCh` 关闭，但 `reconnectLoop` 可能在此之后仍然执行 `wg.Add(2)`

### 问题代码路径

```
Stop() 调用
    ↓
close(stopCh)
    ↓
wg.Wait() 开始等待
    ↓                          ↘
                        reconnectLoop() 正在运行
                               ↓
                        connectInternal() 成功
                               ↓
                        wg.Add(2)  ← 增加了 WaitGroup 计数！
                               ↓
                        启动新的 readLoop/writeLoop
    ↓
wg.Wait() 需要等待额外的 goroutines
    ↓
超时 5 秒
```

## 解决方案

### 1. 添加 `stopped` 原子变量

```go
// client.go
type Client struct {
    // ...
    stopped atomic.Bool // 标识 client 已被永久停止
    wgMu    sync.Mutex  // 保护 wg.Add() 的原子性
}
```

### 2. 修改 Stop() 方法

```go
func (c *Client) Stop() {
    c.stopOnce.Do(func() {
        // 在 wgMu 保护下设置 stopped=true
        // 确保设置后不会有新的 wg.Add() 调用
        c.wgMu.Lock()
        c.stopped.Store(true)
        c.wgMu.Unlock()

        // ... 其余清理逻辑
    })
}
```

### 3. 修改 Start() 方法

```go
func (c *Client) Start() bool {
    c.wgMu.Lock()
    defer c.wgMu.Unlock()

    // 检查 + wg.Add 必须原子执行
    if c.stopped.Load() {
        return false
    }

    c.wg.Add(2)
    go c.readLoop()
    go c.writeLoop()
    return true
}
```

### 4. 修改 readLoop() defer 顺序

```go
func (c *Client) readLoop() {
    defer func() {
        // wg.Done() 必须首先执行，在任何回调之前
        c.wg.Done()

        // ... 其余清理逻辑和回调
    }()
}
```

### 5. 修改 reconnectLoop() 中的 wg.Add()

```go
// 在成功重连后，启动新 loops 之前
c.wgMu.Lock()
if c.stopped.Load() {
    c.wgMu.Unlock()
    // 清理并返回
    return
}
c.wg.Add(2)
c.wgMu.Unlock()

go c.readLoop()
go c.writeLoop()
```

## 验证测试

新增以下测试用例确保没有回归：

- `TestStopDuringReconnect`: 验证在重连过程中调用 `Stop()` 不会卡住
- `TestConcurrentStopAndReconnect`: 验证并发场景下不会 panic 或 hang
- `TestStartAfterStop`: 验证 `Stop()` 后 `Start()` 返回 false
- `TestStopIdempotent`: 验证多次调用 `Stop()` 是安全的

运行测试：
```bash
bazel test //runner/internal/relay/... --test_filter='TestStopDuringReconnect|TestConcurrentStopAndReconnect|TestStartAfterStop|TestStopIdempotent'
```

## 相关文件

- `runner/internal/relay/client.go` - Client 结构体定义
- `runner/internal/relay/client_connection.go` - Connect/Start/Stop 方法
- `runner/internal/relay/client_loops.go` - readLoop/writeLoop 实现
- `runner/internal/relay/client_reconnect.go` - reconnectLoop 实现
- `runner/internal/relay/client_test.go` - 测试用例

## 关键教训

1. **WaitGroup 和信号 channel 之间需要同步**: 仅使用 channel 信号不足以防止竞态，需要额外的锁来保证原子性
2. **defer 执行顺序很重要**: `wg.Done()` 应该在 defer 中尽早执行，不应被其他耗时操作阻塞
3. **"已停止"状态需要显式检查**: 不能仅依赖 channel 关闭，需要原子变量配合锁来确保一致性
