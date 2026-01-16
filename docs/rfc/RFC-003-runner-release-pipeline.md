# RFC-003: Runner 发布流程设计

| 属性 | 值 |
|------|-----|
| **状态** | Accepted |
| **作者** | AgentsMesh Team |
| **创建日期** | 2026-01-16 |
| **目标** | 规范 Runner 多平台发布流程 |

---

## 1. 概述

### 1.1 背景

AgentsMesh Runner 需要支持多平台分发，包括：
- **CLI 版本**：无 GUI，适用于服务器和命令行环境
- **Desktop 版本**：带系统托盘，适用于桌面环境

由于 Desktop 版本需要 CGO 和平台特定的 GUI 库，无法简单地交叉编译，因此需要设计一套完整的多平台构建和发布流程。

### 1.2 发布仓库

- **源码仓库**: GitLab `aio/agentsmesh` (monorepo)
- **发布仓库**: GitHub `AgentsMesh/AgentsMeshRunner`

使用独立的 GitHub 仓库发布，便于用户下载和 Homebrew/Scoop 等包管理器集成。

---

## 2. 构建架构

### 2.1 构建矩阵

| 版本 | 平台 | 架构 | 构建方式 | CGO |
|------|------|------|----------|-----|
| CLI | Linux | amd64, arm64 | GoReleaser (Docker) | ❌ |
| CLI | macOS | amd64, arm64, universal | GoReleaser (Docker) | ❌ |
| CLI | Windows | amd64, arm64 | GoReleaser (Docker) | ❌ |
| Desktop | macOS | universal | Native Runner (shell) | ✅ |
| Desktop | Linux | amd64 | Docker (golang + GTK) | ✅ |
| Desktop | Windows | amd64 | Native Runner (shell) | ✅ |

### 2.2 构建工具

```
┌─────────────────────────────────────────────────────────────┐
│                    GitLab CI/CD Pipeline                    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │  build:cli  │    │build:desktop│    │build:desktop│     │
│  │ (GoReleaser)│    │   :macos    │    │   :linux    │     │
│  │   Docker    │    │ Shell/Xcode │    │   Docker    │     │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘     │
│         │                  │                  │             │
│         ▼                  ▼                  ▼             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │   dist/     │    │package:dmg  │    │package:tar  │     │
│  │ *.tar.gz    │    │   (DMG)     │    │  (tar.gz)   │     │
│  │ *.zip       │    └──────┬──────┘    └──────┬──────┘     │
│  │ *.deb/rpm   │           │                  │             │
│  └──────┬──────┘           │                  │             │
│         │                  │                  │             │
│         └──────────────────┼──────────────────┘             │
│                            ▼                                │
│                   ┌─────────────────┐                       │
│                   │ release:github  │                       │
│                   │ (Upload Assets) │                       │
│                   └─────────────────┘                       │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. 关键配置

### 3.1 GoReleaser 配置 (`runner/.goreleaser.yml`)

```yaml
version: 2
project_name: agentsmesh-runner

builds:
  - id: runner-cli
    main: ./cmd/runner
    binary: runner
    env:
      - CGO_ENABLED=0  # CLI 不需要 CGO
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.buildTime={{.Date}}

archives:
  - id: cli-archive
    builds: [runner-cli]
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
  # macOS Universal Binary 单独打包
  - id: cli-archive-darwin-universal
    builds: [runner-universal]
    name_template: "{{ .ProjectName }}_{{ .Version }}_darwin_all"

universal_binaries:
  - id: runner-universal
    ids: [runner-cli]
    replace: false  # 保留独立架构的构建
    name_template: "runner"

nfpms:
  - id: runner-deb-rpm
    package_name: agentsmesh-runner
    formats: [deb, rpm, apk]
```

### 3.2 GitLab CI 配置 (`runner/.gitlab-ci.yml`)

#### 3.2.1 CLI 构建

```yaml
build:cli:
  stage: build
  tags: [amd64, docker]
  image:
    name: goreleaser/goreleaser:latest
    entrypoint: [""]
  script:
    - cd runner
    # --skip=publish: 只构建，不发布（发布由 release:github 处理）
    - goreleaser release --clean --skip=publish
  artifacts:
    paths:
      - runner/dist/
  rules:
    - if: $CI_COMMIT_TAG
```

#### 3.2.2 macOS Desktop 构建

```yaml
build:desktop:macos:
  stage: build
  tags: [macos, xcode]  # 需要 macOS 原生 Runner
  before_script:
    - export PATH="/opt/homebrew/opt/go@1.24/bin:/opt/homebrew/bin:$PATH"
    # Shell executor 需要显式设置 VERSION
    - export VERSION="${CI_COMMIT_TAG:-dev}"
  script:
    - cd runner
    - export VERSION="${CI_COMMIT_TAG:-dev}"
    # 构建 arm64 (原生)
    - CGO_ENABLED=1 GOARCH=arm64 go build -tags desktop \
        -ldflags "-s -w -X main.version=$VERSION" \
        -o runner-desktop-darwin-arm64 ./cmd/runner
    # 构建 amd64 (交叉编译)
    - CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -tags desktop \
        -ldflags "-s -w -X main.version=$VERSION" \
        -o runner-desktop-darwin-amd64 ./cmd/runner
    # 创建 Universal Binary
    - lipo -create -output runner-desktop-darwin-universal \
        runner-desktop-darwin-amd64 runner-desktop-darwin-arm64
```

#### 3.2.3 DMG 打包

```yaml
package:macos-dmg:
  stage: package
  tags: [macos, xcode]
  script:
    - cd runner
    - export VERSION="${CI_COMMIT_TAG:-dev}"
    # 创建 .app bundle
    - mkdir -p "AgentsMesh Runner.app/Contents/MacOS"
    - cp runner-desktop-darwin-universal "AgentsMesh Runner.app/Contents/MacOS/runner-desktop"
    # 创建 DMG
    - DMG_NAME="AgentsMesh-Runner-${VERSION}-darwin-universal.dmg"
    - create-dmg --volname "AgentsMesh Runner" "$DMG_NAME" "AgentsMesh Runner.app" || true
    # 处理 create-dmg AppleScript 超时留下的临时文件
    - |
      if [ ! -f "$DMG_NAME" ]; then
        TEMP_DMG=$(ls rw.*.dmg 2>/dev/null | head -1)
        if [ -n "$TEMP_DMG" ]; then
          mv "$TEMP_DMG" "$DMG_NAME"
        fi
      fi
```

---

## 4. 注意事项

### 4.1 Shell Executor 变量问题

**问题**：macOS/Windows 使用 shell executor，GitLab CI 的全局变量（如 `VERSION`）可能不会自动展开。

**解决方案**：在 `script` 部分显式设置：
```yaml
script:
  - export VERSION="${CI_COMMIT_TAG:-dev}"
  - echo "Building with VERSION=$VERSION"
```

### 4.2 GoReleaser --snapshot vs --skip=publish

| 参数 | 版本号示例 | 用途 |
|------|-----------|------|
| `--snapshot` | `0.2.0-SNAPSHOT-abc1234` | 本地测试 |
| `--skip=publish` | `0.2.0` | CI 构建（干净版本号） |

**正式发布必须使用 `--skip=publish`**，否则版本号会带有 SNAPSHOT 后缀。

### 4.3 create-dmg AppleScript 超时

**问题**：`create-dmg` 在 CI 环境中可能因 AppleScript 超时而失败，但会留下 `rw.xxxxx.*.dmg` 临时文件。

**解决方案**：
```bash
create-dmg ... || true  # 允许失败
# 重命名临时文件
if [ ! -f "$DMG_NAME" ]; then
  TEMP_DMG=$(ls rw.*.dmg 2>/dev/null | head -1)
  [ -n "$TEMP_DMG" ] && mv "$TEMP_DMG" "$DMG_NAME"
fi
```

### 4.4 macOS 交叉编译

在 ARM64 Mac 上可以交叉编译 AMD64：
```bash
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -tags desktop ...
```

然后使用 `lipo` 创建 Universal Binary：
```bash
lipo -create -output runner-universal runner-amd64 runner-arm64
```

### 4.5 可选构建

Linux/Windows Desktop 构建依赖特定环境（GTK、Windows Runner），设置为可选以避免阻塞发布：

```yaml
build:desktop:linux:
  rules:
    - if: $CI_COMMIT_TAG && $DESKTOP_BUILDS == "true"

release:github:
  needs:
    - job: package:linux-archive
      optional: true  # 可选依赖
```

---

## 5. 发布流程

### 5.1 创建新版本

```bash
# 1. 确保代码已合并到 main
git checkout main
git pull origin main

# 2. 创建 tag
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0

# 3. 等待 CI Pipeline 完成
# 4. 检查 GitHub Release: https://github.com/AgentsMesh/AgentsMeshRunner/releases
```

### 5.2 重新发布（修复问题后）

```bash
# 删除旧 tag
git tag -d v0.2.0
git push origin :refs/tags/v0.2.0

# 重新创建 tag
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0
```

### 5.3 发布产物

| 文件 | 说明 |
|------|------|
| `agentsmesh-runner_X.Y.Z_darwin_all.tar.gz` | macOS CLI (Universal) |
| `agentsmesh-runner_X.Y.Z_darwin_amd64.tar.gz` | macOS CLI (Intel) |
| `agentsmesh-runner_X.Y.Z_darwin_arm64.tar.gz` | macOS CLI (Apple Silicon) |
| `agentsmesh-runner_X.Y.Z_linux_amd64.tar.gz` | Linux CLI (x64) |
| `agentsmesh-runner_X.Y.Z_linux_arm64.tar.gz` | Linux CLI (ARM64) |
| `agentsmesh-runner_X.Y.Z_linux_amd64.deb` | Debian/Ubuntu 包 |
| `agentsmesh-runner_X.Y.Z_linux_amd64.rpm` | RHEL/CentOS/Fedora 包 |
| `agentsmesh-runner_X.Y.Z_linux_amd64.apk` | Alpine 包 |
| `agentsmesh-runner_X.Y.Z_windows_amd64.zip` | Windows CLI (x64) |
| `agentsmesh-runner_X.Y.Z_windows_arm64.zip` | Windows CLI (ARM64) |
| `AgentsMesh-Runner-vX.Y.Z-darwin-universal.dmg` | macOS Desktop (Universal) |

---

## 6. Runner 配置要求

### 6.1 macOS Runner

- **Tags**: `[macos, xcode]`
- **Executor**: Shell
- **要求**:
  - Go 1.24+ (`brew install go@1.24`)
  - Xcode Command Line Tools
  - create-dmg (`brew install create-dmg`)

### 6.2 Docker Runner

- **Tags**: `[amd64, docker]`
- **Executor**: Docker
- **要求**:
  - Docker daemon
  - 网络访问 Docker Hub 和 GitHub API

### 6.3 Windows Runner (可选)

- **Tags**: `[windows]`
- **Executor**: Shell (PowerShell)
- **要求**:
  - Go 1.24+
  - CGO 编译环境 (MinGW-w64)

---

## 7. 故障排查

### 7.1 版本号带 SNAPSHOT

**原因**：使用了 `goreleaser release --snapshot`

**解决**：改用 `goreleaser release --clean --skip=publish`

### 7.2 macOS DMG 文件名不正确

**原因**：Shell executor 中 `$VERSION` 变量为空

**解决**：在 script 中显式设置 `export VERSION="${CI_COMMIT_TAG:-dev}"`

### 7.3 GitHub Release 已存在 (422 错误)

**原因**：重复发布同一 tag

**解决**：release:github job 会自动检测并删除旧 release 后重新创建

### 7.4 Docker 镜像拉取失败

**原因**：网络问题或 Docker Hub 限流

**解决**：重试 job，或配置镜像代理

---

## 8. 后续优化

- [ ] 添加 Homebrew Tap 自动更新
- [ ] 添加 Scoop Bucket 自动更新
- [ ] 添加代码签名 (macOS notarization, Windows Authenticode)
- [ ] 添加 SBOM (Software Bill of Materials) 生成
- [ ] 添加 Cosign 签名验证
