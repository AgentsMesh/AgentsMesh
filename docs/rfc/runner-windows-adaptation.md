# Runner Windows 适配问题清单

> 测试环境：Windows 11 ARM64 (Parallels VM)，Go 1.26.0，Git 2.53.0
>
> 基准分支：`main` @ `1db26865`
>
> 日期：2026-03-06

## 概述

在 Windows 上执行 `go build ./cmd/runner`（通过）和 `go test ./...`（部分失败）后，
发现以下需要适配的问题。

**测试结果总览：**

| 模块 | 状态 | 失败数 |
|------|------|--------|
| workspace | FAIL | 6 |
| updater | FAIL | 4 (含 2 panic) |
| terminal (core) | FAIL | 15 |
| terminal/aggregator | PASS | 0 |
| terminal/detector | FAIL | 1 |
| autopilot | FAIL | 17 |
| mcp | FAIL | 9 |
| cache | FAIL | 4 (含 1 panic) |
| clipboard | FAIL | 8 |
| envpath | FAIL | 1 |
| runner | FAIL | 1 |
| client, config, lifecycle, logger, monitor, relay, safego, terminal/vt | PASS | 0 |

按根因分类为 **5 大类问题**，共涉及 **12 个模块**，**66 个失败测试**。

---

## 类别 A：Shell / 可执行文件硬编码为 Unix 命令（高 — 功能阻断）

这是最核心的问题。Runner 在多处硬编码了 Unix shell 和命令，Windows 上完全不可用。

### A1. workspace — Shell 硬编码为 `/bin/sh`

**涉及文件：**
- `runner/internal/workspace/preparer.go:193`

**现象：**
`ScriptPreparationStep.Execute()` 中硬编码了 `/bin/sh -c`：
```go
cmd := exec.CommandContext(ctx, "/bin/sh", "-c", s.script)
```
Windows 上不存在 `/bin/sh`，导致所有脚本执行类测试失败：
```
exec: "/bin/sh": executable file not found in %PATH%
```

**影响的测试（3 个）：**
- `TestPreparerPrepareSuccess`
- `TestPreparerMultipleSteps`
- `TestScriptPreparationStepExecuteWithEnvVars`

**建议修复：**
根据 `runtime.GOOS` 选择 shell：
```go
if runtime.GOOS == "windows" {
    cmd = exec.CommandContext(ctx, "cmd", "/C", s.script)
} else {
    cmd = exec.CommandContext(ctx, "/bin/sh", "-c", s.script)
}
```

### A2. terminal — PTY 测试依赖 Unix 命令（`sh`, `echo`, `cat`, `sleep`, `true`, `false`）

**涉及文件：**
- `runner/internal/terminal/terminal_pty_test.go`
- `runner/internal/terminal/terminal_handler_test.go`

**现象：**
测试中直接使用 Unix 命令作为 PTY 进程启动参数：
```
exec: "sh": executable file not found in %PATH%
exec: "echo": executable file not found in %PATH%
exec: "sleep": executable file not found in %PATH%
exec: "cat": executable file not found in %PATH%
exec: "true": executable file not found in %PATH%
exec: "false": executable file not found in %PATH%
```

**影响的测试（15 个）：**
- `TestSetOutputHandler`, `TestSetExitHandler`, `TestSetOutputHandlerNil`, `TestSetExitHandlerNil`
- `TestSetHandlersBeforeStart`, `TestTerminalRedrawClosed`, `TestTerminalRedrawSuccess`
- `TestTerminalStartSuccess`, `TestTerminalWriteSuccess`, `TestTerminalResizeSuccess`
- `TestTerminalPIDRunning`, `TestTerminalStopRunning`
- `TestTerminalWriteClosed`, `TestTerminalResizeClosed`, `TestTerminalExitCode`

**建议修复：**
在测试中使用跨平台命令映射：
```go
func testShell() string {
    if runtime.GOOS == "windows" { return "cmd" }
    return "sh"
}
func testSleep(seconds string) []string {
    if runtime.GOOS == "windows" { return []string{"cmd", "/C", "timeout /t " + seconds + " /nobreak >nul"} }
    return []string{"sleep", seconds}
}
```

### A3. autopilot — 测试创建 Bash 脚本作为 mock agent

**涉及文件：**
- `runner/internal/autopilot/autopilot_controller_decision_test.go:52-56, 116-120, 170-174, 234-238, 337-340, 405-408`
- `runner/internal/autopilot/control_runner_test.go:145, 181-184, 220-223`

**现象：**
测试动态创建 `#!/bin/bash` 脚本作为 mock agent，Windows 无法执行：
```go
script := `#!/bin/bash
echo '{"decision": "completed", ...}'`
os.WriteFile(scriptPath, []byte(script), 0755)
```
错误：`executable file not found in %PATH%`

**影响的测试（17 个）：**
- `TestAutopilotController_HandleDecision_Completed/NeedHumanHelp/GiveUp/Continue`
- `TestControlRunner_StartControlProcess_WithLogger/Success/Timeout/LongOutputTruncation`
- `TestControlRunner_ResumeControlProcess_WithLogger/Success/LongOutputTruncation`
- `TestControlRunner_RunControlProcess_Start/Resume`

**建议修复：**
Windows 上生成 `.bat` 脚本替代 bash 脚本：
```go
func writeMockScript(t *testing.T, dir, name, content string) string {
    if runtime.GOOS == "windows" {
        path := filepath.Join(dir, name+".bat")
        bat := fmt.Sprintf("@echo off\r\necho %s", content)
        os.WriteFile(path, []byte(bat), 0755)
        return path
    }
    path := filepath.Join(dir, name)
    script := fmt.Sprintf("#!/bin/bash\necho '%s'", content)
    os.WriteFile(path, []byte(script), 0755)
    return path
}
```

### A4. mcp — MCP Server 启动依赖 Python shebang

**涉及文件：**
- `runner/internal/mcp/mock_server_test.go:14, 107`
- `runner/internal/mcp/mock_server_error_test.go:13, 84, 154, 163, 358`

**现象：**
Mock MCP server 使用 `#!/usr/bin/env python3` shebang，Windows 不识别 shebang 行。
测试超时后报 `context deadline exceeded`。

**影响的测试（9 个）：**
- `TestCallToolMCPErrorResponse`, `TestCallToolIsErrorResponse`
- `TestReadResourceErrorResponse`, `TestReadResourceEmptyContentsError`
- `TestCallToolEmptyIsError`
- `TestServerWithMockMCP`, `TestServerCallToolWithMockMCP`, `TestServerReadResourceWithMockMCP`
- `TestManagerWithMockMCP`

**建议修复：**
在启动 MCP server 时显式用 `python3` 命令（而非依赖 shebang），
或在 Windows 上使用 `py -3` 命令。代码中 `Command: "python3"` 已设置，
但实际启动逻辑可能未使用该字段。需排查 MCP server 的实际启动方式。

### A5. runner — Pod 创建使用 Unix 命令

**涉及文件：**
- `runner/internal/runner/message_handler_pod_create_test.go:42`

**现象：**
`TestOnCreatePodSuccess` 创建 Pod 时使用 `sleep 10` 命令，Windows 无此命令。
```
exec: "sleep": executable file not found in %PATH%
```

**影响的测试（1 个）：**
- `TestOnCreatePodSuccess`

**建议修复：**
同 A2，测试中 `sleep` 改为跨平台命令。

---

## 类别 B：路径分隔符 / 路径格式硬编码为 Unix 风格（中 — 功能隐患）

### B1. workspace — PATH 分隔符硬编码为 `:`

**涉及文件：**
- `runner/internal/workspace/preparer.go:239-264` — `addToolPaths()`

**现象：**
`addToolPaths()` 仅处理了 `darwin` 和 `linux`，缺少 `windows` 分支。
PATH 拼接使用了 Unix 分隔符 `:`，Windows 下应为 `;`。

**建议修复：**
使用 `os.PathListSeparator` 替代硬编码分隔符，并增加 Windows 路径分支：
```go
sep := string(os.PathListSeparator)
env[i] = "PATH=" + extraPaths + sep + currentPath
```

### B2. cache — 路径使用 `filepath.Join` 后对比 Unix 风格字符串

**涉及文件：**
- `runner/internal/cache/downloader_test.go:76`

**现象：**
`filepath.Join` 在 Windows 下返回 `\` 分隔路径，但测试期望 `/` 分隔：
```
expected: "/home/user/sandbox/skills/my-skill"
actual  : "\home\user\sandbox\skills\my-skill"
```

**影响的测试（3 个）：**
- `TestResolveResourcePath/replaces_sandbox_root_path`
- `TestResolveResourcePath/replaces_sandbox_work_dir`
- `TestResolveResourcePath/replaces_both_templates`

**建议修复：**
测试中使用 `filepath.FromSlash()` 转换期望路径，或在业务逻辑中统一使用 `/`。

### B3. clipboard — 路径硬编码对比 Unix 风格

**涉及文件：**
- `runner/internal/clipboard/shim_test.go:259, 267`

**现象：**
```
ShimBinDir: got "\\tmp\\sandbox\\.clipboard-shim\\bin", want "/tmp/sandbox/.clipboard-shim/bin"
dataDir: got "\\tmp\\sandbox\\.clipboard-shim\\data", want "/tmp/sandbox/.clipboard-shim/data"
```

**影响的测试（2 个）：**
- `TestShimBinDir`
- `TestDataDir`

**建议修复：**
同 B2，使用 `filepath.FromSlash()` 统一路径风格。

---

## 类别 C：Unix 特有 API / 权限模型（中 — 功能 + 测试）

### C1. updater — `syscall.Signal(0)` 不支持 Windows

**涉及文件：**
- `runner/internal/updater/graceful_apply.go:141-148` — `DefaultHealthChecker()`

**现象：**
健康检查使用 `syscall.Signal(0)` 探测进程存活，Windows 不支持：
```
"not supported by windows"
```

**影响的测试（2 个）：**
- `TestDefaultHealthChecker_ProcessRunning`
- `TestDefaultHealthChecker_ProcessNotFound`

**建议修复：**
使用 build tags 分离平台实现：
- `health_check_unix.go` (//go:build !windows): 保持 `Signal(0)`
- `health_check_windows.go` (//go:build windows): 使用 `windows.OpenProcess()`

### C2. updater — `os.Rename` 行为差异导致 panic

**涉及文件：**
- `runner/internal/updater/updater_options_test.go:145-160`

**现象：**
测试创建目录覆盖可执行文件路径，期望 `Rollback()` 返回错误。
Windows 下 `os.Rename` 行为不同，`err` 为 `nil`，后续 nil pointer dereference。

**影响的测试（1 个 + panic）：**
- `TestUpdater_Rollback_AtomicReplaceError` — panic

**建议修复：**
审查 `atomicReplace` 实现，确保 Windows 下正确返回错误；
或调整测试 setup 在 Windows 上使用不同的错误触发方式。

### C3. workspace / clipboard — `os.Chmod(0000)` 在 NTFS 上无效

**涉及文件：**
- `runner/internal/workspace/manager_test.go:113`
- `runner/internal/workspace/cleanup_worktree_test.go:100`
- `runner/internal/workspace/list_worktree_test.go:60`
- `runner/internal/clipboard/shim_test.go:66, 86, 105, 222`

**现象：**
通过 `os.Chmod(dir, 0000)` 创建不可读目录来触发错误路径，
但 Windows NTFS 不支持 Unix 权限位，`Chmod` 是 no-op。

**影响的测试（7 个）：**
- workspace: `TestNewManagerError`, `TestCleanupOldWorktreesReadDirError`, `TestListWorktreesReadError`
- clipboard: `TestSetupShims_CannotCreateBinDir`, `TestSetupShims_CannotCreateDataDir`, `TestSetupShims_CannotWriteXclip`, `TestWriteImage_CannotCreateDataDir`

**建议修复：**
- 短期：`if runtime.GOOS == "windows" { t.Skip("...") }`
- 长期：使用 Windows ACL API (`icacls`) 或替代的错误注入方式

### C4. updater — 可执行文件缺 `.exe` 后缀

**涉及文件：**
- `runner/internal/updater/graceful_restart_mock_test.go:37`

**现象：**
测试创建临时二进制文件但不带 `.exe` 后缀，Windows 上 `exec.Command` 找不到。

**影响的测试（1 个）：**
- `TestDefaultRestartFunc_Success`

**建议修复：**
```go
suffix := ""
if runtime.GOOS == "windows" { suffix = ".exe" }
tmpFile := filepath.Join(t.TempDir(), "runner"+suffix)
```

---

## 类别 D：Unix 环境假设（中 — 功能 + 测试）

### D1. envpath — 测试假设 PATH 包含 `/usr/bin`

**涉及文件：**
- `runner/internal/envpath/envpath_test.go:20`

**现象：**
`TestResolveLoginShellPATH_ContainsStandardDirs` 期望 PATH 包含 `/usr/bin`，
但 Windows PATH 不包含 Unix 路径。

**影响的测试（1 个）：**
- `TestResolveLoginShellPATH_ContainsStandardDirs`

**建议修复：**
按 `runtime.GOOS` 检测不同的标准目录：
- Unix: 检查 `/usr/bin`
- Windows: 检查 `C:\WINDOWS\system32`

### D2. clipboard — shim 脚本是 shell 脚本，Windows 不可执行

**涉及文件：**
- `runner/internal/clipboard/backend_test.go:292`
- `runner/internal/clipboard/shim_test.go:26`

**现象：**
剪贴板 shim 创建的是 Unix shell 脚本（`xclip`, `osascript`），
Windows 上无法标记为可执行（无 chmod +x 效果）。

**影响的测试（2 个）：**
- `TestShimBackend_Setup`（"shim xclip not executable"）
- `TestSetupShims`（"shim xclip not executable"）

**建议修复：**
Windows 上跳过 Unix-only 的 shim 测试，或提供 `.bat` / `.ps1` 版本的 shim。

### D3. cache — `/dev/null` 路径在 Windows 上可被创建

**涉及文件：**
- `runner/internal/cache/skill_cache_test.go:173-175`

**现象：**
`TestNewSkillCacheManager_InvalidPath` 使用 `/dev/null/impossible/cache/dir` 作为无效路径，
期望 `NewSkillCacheManager` 返回错误。但 Windows 上该路径格式合法（`\dev\null\...`），
不返回错误，后续 nil pointer dereference。

**影响的测试（1 个 + panic）：**
- `TestNewSkillCacheManager_InvalidPath` — panic

**建议修复：**
使用 Windows 上也无效的路径，或按平台选择：
```go
invalidPath := "/dev/null/impossible"
if runtime.GOOS == "windows" {
    invalidPath = `\\.\NUL\impossible\cache\dir`
}
```

---

## 类别 E：Timing 敏感（低 — 环境相关）

### E1. terminal/detector — 窗口重置 timing 竞争

**涉及文件：**
- `runner/internal/terminal/detector/output_activity_detector_test.go:193`

**现象：**
`TestOutputActivityDetector_WindowReset` 依赖 `time.Sleep(60ms)` 来触发窗口过期，
在 Windows VM 上由于定时器精度问题（默认 ~15.6ms 精度），`GetOutputRate()` 返回 0。

**影响的测试（1 个）：**
- `TestOutputActivityDetector_WindowReset`

### E2. terminal/detector — OSC 信号 timing

**涉及文件：**
- `runner/internal/terminal/detector/multi_signal_detector_test.go`

**影响的测试（1 个）：**
- `TestSignal_OSCOverriddenByOutput`

**建议修复：**
- 增大 timing 容差（60ms → 200ms）
- 或使用可控时钟 (mock clock) 替代 `time.Sleep`

---

## 总结

| 类别 | 问题描述 | 涉及模块 | 失败测试数 | 严重程度 |
|------|----------|----------|------------|----------|
| **A** | Shell/命令硬编码为 Unix | workspace, terminal, autopilot, mcp, runner | **45** | **高** |
| **B** | 路径分隔符硬编码 | workspace, cache, clipboard | **5** | 中 |
| **C** | Unix 特有 API/权限 | updater, workspace, clipboard | **11** | 中 |
| **D** | Unix 环境假设 | envpath, clipboard, cache | **4** | 中 |
| **E** | Timing 敏感 | terminal/detector | **2** | 低 |
| | **合计** | **12 模块** | **67** | |

## 修复优先级建议

### P0 — 功能阻断（先修，解除 67% 失败）
1. **A1**: `preparer.go` Shell 选择 — 影响 Runner 核心功能
2. **C1**: `graceful_apply.go` 进程健康检查 — 影响自更新功能
3. **B1**: `preparer.go` PATH 分隔符 — 影响脚本执行环境

### P1 — 测试适配（解除大部分测试失败）
4. **A2**: terminal PTY 测试命令映射
5. **A3**: autopilot mock agent 脚本
6. **A4**: MCP mock server 启动方式
7. **A5**: runner Pod 创建测试命令
8. **B2/B3**: 路径对比使用 `filepath.FromSlash()`
9. **C3**: `os.Chmod` 错误路径测试 — `t.Skip` on Windows
10. **C4**: `.exe` 后缀

### P2 — 环境适配
11. **C2**: `atomicReplace` Windows 行为审查
12. **D1/D2/D3**: 环境假设修正
13. **E1/E2**: timing 容差调整
