# AgentsMesh Runner

[![Release](https://img.shields.io/github/v/release/AgentsMesh/AgentsMesh?style=flat-square)](https://github.com/AgentsMesh/AgentsMesh/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/anthropics/agentsmesh/runner?style=flat-square)](https://goreportcard.com/report/github.com/anthropics/agentsmesh/runner)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE)

AgentsMesh Runner is a lightweight agent that connects to the AgentsMesh server and executes AI agent tasks in isolated terminal environments.

## Features

- 🚀 **Multi-mode operation**: CLI or system service
- 🔒 **Secure execution**: Isolated terminal environments for each task
- 🌐 **Cross-platform**: macOS, Linux, Windows support
- 📊 **Web console**: Built-in status monitoring and log viewer
- 🔄 **Auto-reconnect**: Resilient connection to AgentsMesh server

## Installation

### macOS / Linux (one-line)

```bash
curl -fsSL https://agentsmesh.ai/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://agentsmesh.ai/install.ps1 | iex
```

### macOS (Homebrew)

```bash
brew tap agentsmesh/tap https://github.com/AgentsMesh/BrewCask
brew install agentsmesh/tap/agentsmesh-runner
```

All binaries are available on the [Releases](https://github.com/AgentsMesh/AgentsMesh/releases/latest) page (tar.gz, deb, rpm, zip).

## Quick Start

### 1. Register the runner

Get a registration token from your AgentsMesh dashboard, then:

```bash
# Global: https://agentsmesh.ai (or your own server address)
agentsmesh-runner register --server <SERVER_URL> --token YOUR_TOKEN
```

### 2. Start the runner

**CLI mode (foreground):**

```bash
agentsmesh-runner run
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
  webconsole  Open the web console in browser
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

When using the web console command, a local web UI is available at:

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

# Build (no CGO required)
make build

# Build for all platforms
make build-all
```

## Release

Releases are published to [AgentsMesh/AgentsMesh](https://github.com/AgentsMesh/AgentsMesh).

To create a new release:

```bash
# Tag a new version
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

The CI pipeline will automatically:
- Build CLI binaries for all platforms (via GoReleaser)
- Publish to GitHub Releases
- Update Homebrew formula

## License

MIT License - see [LICENSE](LICENSE) for details.

## Links

- [AgentsMesh](https://agentsmesh.ai) - Main product website
- [Documentation](https://agentsmesh.ai/docs/runner) - Full documentation
- [Releases](https://github.com/AgentsMesh/AgentsMesh/releases) - Download binaries
