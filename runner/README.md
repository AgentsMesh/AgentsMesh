# AgentsMesh Runner

[![Release](https://img.shields.io/github/v/release/AgentsMesh/AgentsMeshRunner?style=flat-square)](https://github.com/AgentsMesh/AgentsMeshRunner/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/anthropics/agentsmesh/runner?style=flat-square)](https://goreportcard.com/report/github.com/anthropics/agentsmesh/runner)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE)

AgentsMesh Runner is a lightweight agent that connects to the AgentsMesh server and executes AI agent tasks in isolated terminal environments.

## Features

- 🚀 **Multi-mode operation**: CLI, system service, or desktop tray
- 🔒 **Secure execution**: Isolated terminal environments for each task
- 🌐 **Cross-platform**: macOS, Linux, Windows support
- 📊 **Web console**: Built-in status monitoring and log viewer
- 🔄 **Auto-reconnect**: Resilient connection to AgentsMesh server

## Installation

### macOS (Homebrew)

```bash
brew tap agentsmesh/tap https://github.com/AgentsMesh/BrewCask
brew install agentsmesh/tap/agentsmesh-runner
```

### macOS/Linux (Direct download)

```bash
# macOS (Universal binary - Intel & Apple Silicon)
curl -fsSL https://github.com/AgentsMesh/AgentsMeshRunner/releases/latest/download/agentsmesh-runner_0.2.1_darwin_all.tar.gz | tar xz
sudo mv agentsmesh-runner /usr/local/bin/

# Linux x86_64
curl -fsSL https://github.com/AgentsMesh/AgentsMeshRunner/releases/latest/download/agentsmesh-runner_0.2.1_linux_amd64.tar.gz | tar xz
sudo mv agentsmesh-runner /usr/local/bin/

# Linux ARM64
curl -fsSL https://github.com/AgentsMesh/AgentsMeshRunner/releases/latest/download/agentsmesh-runner_0.2.1_linux_arm64.tar.gz | tar xz
sudo mv agentsmesh-runner /usr/local/bin/
```

> **Note**: Replace `0.2.1` with the [latest version](https://github.com/AgentsMesh/AgentsMeshRunner/releases/latest).

### Linux (Debian/Ubuntu)

```bash
# Download the latest .deb package (x86_64)
VERSION=$(curl -s https://api.github.com/repos/AgentsMesh/AgentsMeshRunner/releases/latest | grep tag_name | cut -d '"' -f 4 | sed 's/v//')
wget https://github.com/AgentsMesh/AgentsMeshRunner/releases/download/v${VERSION}/agentsmesh-runner_${VERSION}_linux_amd64.deb
sudo dpkg -i agentsmesh-runner_${VERSION}_linux_amd64.deb
```

### Linux (RHEL/CentOS/Fedora)

```bash
# Download the latest .rpm package (x86_64)
VERSION=$(curl -s https://api.github.com/repos/AgentsMesh/AgentsMeshRunner/releases/latest | grep tag_name | cut -d '"' -f 4 | sed 's/v//')
wget https://github.com/AgentsMesh/AgentsMeshRunner/releases/download/v${VERSION}/agentsmesh-runner_${VERSION}_linux_amd64.rpm
sudo rpm -i agentsmesh-runner_${VERSION}_linux_amd64.rpm
```

### Windows

Download the latest `.zip` file from [Releases](https://github.com/AgentsMesh/AgentsMeshRunner/releases/latest), extract, and add to your PATH.

Or using Scoop:

```powershell
scoop bucket add agentsmesh https://github.com/AgentsMesh/scoop-bucket
scoop install agentsmesh-runner
```

## Quick Start

### 1. Register the runner

Get a registration token from your AgentsMesh dashboard, then:

```bash
# Global: https://api.agentsmesh.ai (or your own server address)
agentsmesh-runner register --server <SERVER_URL> --token YOUR_TOKEN
```

### 2. Start the runner

**CLI mode (foreground):**

```bash
agentsmesh-runner run
```

**Desktop mode (with system tray):**

```bash
agentsmesh-runner desktop
```

**System service:**

```bash
# Install as service
sudo agentsmesh-runner service install

# Start service
sudo agentsmesh-runner service start

# Check status
agentsmesh-runner service status
```

## Usage

```
AgentsMesh Runner

Usage:
  agentsmesh-runner <command> [options]

Commands:
  register    Register this runner with the AgentsMesh server
  run         Start the runner in CLI mode
  desktop     Start runner in desktop mode with system tray
  service     Manage runner as a system service
  version     Show version information
  help        Show this help message

Use "agentsmesh-runner <command> --help" for more information about a command.
```

## Configuration

Configuration is stored in `~/.agentsmesh/config.yaml` after registration:

```yaml
server_url: <SERVER_URL>  # Your AgentsMesh server address
node_id: my-runner
max_concurrent_pods: 5
workspace_root: /tmp/agentsmesh-workspace
default_agent: claude-code
log_level: info
```

## Web Console

When running in desktop mode, a local web console is available at:

```
http://127.0.0.1:19080
```

Features:
- Real-time status monitoring
- Active pods and uptime tracking
- Configuration viewer
- Live log streaming

## Building from Source

```bash
# Clone the repository
git clone https://github.com/anthropics/agentsmesh.git
cd agentsmesh/runner

# Build CLI version (no CGO required)
make build

# Build with desktop support (requires CGO)
make build-desktop

# Build for all platforms
make build-all
```

## Release

Releases are published to [AgentsMesh/AgentsMeshRunner](https://github.com/AgentsMesh/AgentsMeshRunner).

To create a new release:

```bash
# Tag a new version
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

The GitLab CI pipeline will automatically:
- Build CLI binaries for all platforms (via GoReleaser)
- Build Desktop versions (macOS DMG with system tray support)
- Publish to GitHub Releases
- Update Homebrew formula

## License

MIT License - see [LICENSE](LICENSE) for details.

## Links

- [AgentsMesh](https://agentsmesh.ai) - Main product website
- [Documentation](https://agentsmesh.ai/docs/runner) - Full documentation
- [Releases](https://github.com/AgentsMesh/AgentsMeshRunner/releases) - Download binaries
