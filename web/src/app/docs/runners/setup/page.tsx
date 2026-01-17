export default function RunnerSetupPage() {
  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">Runner Setup</h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        Runners are the execution environments for AI agent Pods. Set up a
        runner on any machine with Git and your preferred development tools
        installed.
      </p>

      {/* Requirements */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">System Requirements</h2>
        <ul className="list-disc list-inside text-muted-foreground space-y-2">
          <li>Linux, macOS, or Windows (WSL2 recommended)</li>
          <li>Git installed and configured</li>
          <li>Docker (optional, for containerized agents)</li>
          <li>Network access to AgentsMesh server</li>
          <li>At least 4GB RAM (8GB+ recommended for multiple Pods)</li>
        </ul>
      </section>

      {/* Quick Install */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Quick Installation</h2>

        <h3 className="text-lg font-medium mb-2 mt-6">One-Line Install (Recommended)</h3>
        <div className="bg-[#1a1a1a] rounded-lg p-4 font-mono text-sm overflow-x-auto">
          <pre className="text-green-400">{`# macOS / Linux
curl -fsSL https://agentsmesh.ai/install.sh | sh

# Windows (PowerShell)
irm https://agentsmesh.ai/install.ps1 | iex`}</pre>
        </div>

        <h3 className="text-lg font-medium mb-2 mt-6">macOS (Homebrew)</h3>
        <div className="bg-[#1a1a1a] rounded-lg p-4 font-mono text-sm overflow-x-auto">
          <pre className="text-green-400">{`# Add AgentsMesh tap
brew tap agentsmesh/tap https://github.com/AgentsMesh/BrewCask

# Install runner
brew install agentsmesh/tap/agentsmesh-runner`}</pre>
        </div>

        <h3 className="text-lg font-medium mb-2 mt-6">Linux (Package Manager)</h3>
        <div className="bg-[#1a1a1a] rounded-lg p-4 font-mono text-sm overflow-x-auto">
          <pre className="text-green-400">{`# Debian/Ubuntu
VERSION=$(curl -s https://api.github.com/repos/AgentsMesh/AgentsMeshRunner/releases/latest | grep tag_name | cut -d '"' -f 4 | sed 's/v//')
wget https://github.com/AgentsMesh/AgentsMeshRunner/releases/download/v\${VERSION}/agentsmesh-runner_\${VERSION}_linux_amd64.deb
sudo dpkg -i agentsmesh-runner_\${VERSION}_linux_amd64.deb

# RHEL/CentOS/Fedora
VERSION=$(curl -s https://api.github.com/repos/AgentsMesh/AgentsMeshRunner/releases/latest | grep tag_name | cut -d '"' -f 4 | sed 's/v//')
wget https://github.com/AgentsMesh/AgentsMeshRunner/releases/download/v\${VERSION}/agentsmesh-runner_\${VERSION}_linux_amd64.rpm
sudo rpm -i agentsmesh-runner_\${VERSION}_linux_amd64.rpm`}</pre>
        </div>

        <h3 className="text-lg font-medium mb-2 mt-6">After Installation</h3>
        <div className="bg-[#1a1a1a] rounded-lg p-4 font-mono text-sm overflow-x-auto">
          <pre className="text-green-400">{`# Register with your server
runner register --server https://api.agentsmesh.ai --token <YOUR_TOKEN>

# Start the runner
runner run`}</pre>
        </div>

        <p className="text-sm text-muted-foreground mt-4">
          Get your registration token from{" "}
          <strong>Settings → Runners → Create Token</strong> in the AgentsMesh
          web interface.
        </p>
      </section>

      {/* Docker Installation */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Docker Installation</h2>
        <div className="bg-[#1a1a1a] rounded-lg p-4 font-mono text-sm overflow-x-auto">
          <pre className="text-green-400">{`# Run with Docker
docker run -d \\
  --name agentsmesh-runner \\
  -e AGENTSMESH_TOKEN=<YOUR_TOKEN> \\
  -e AGENTSMESH_URL=https://api.agentsmesh.dev \\
  -v /var/run/docker.sock:/var/run/docker.sock \\
  -v ~/.ssh:/root/.ssh:ro \\
  agentsmesh/runner:latest`}</pre>
        </div>
      </section>

      {/* Docker Compose */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Docker Compose</h2>
        <div className="bg-[#1a1a1a] rounded-lg p-4 font-mono text-sm overflow-x-auto">
          <pre className="text-green-400">{`# docker-compose.yml
version: '3.8'
services:
  runner:
    image: agentsmesh/runner:latest
    container_name: agentsmesh-runner
    restart: unless-stopped
    environment:
      - AGENTSMESH_TOKEN=\${AGENTSMESH_TOKEN}
      - AGENTSMESH_URL=\${AGENTSMESH_URL:-https://api.agentsmesh.dev}
      - MAX_CONCURRENT_PODS=5
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ~/.ssh:/root/.ssh:ro
      - runner-data:/data
volumes:
  runner-data:`}</pre>
        </div>
      </section>

      {/* Environment Variables */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Environment Variables</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">
                  Variable
                </th>
                <th className="text-left p-3 border-b border-border">
                  Description
                </th>
                <th className="text-left p-3 border-b border-border">
                  Default
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border font-mono text-xs">
                  AGENTSMESH_TOKEN
                </td>
                <td className="p-3 border-b border-border">
                  Registration token (required)
                </td>
                <td className="p-3 border-b border-border">-</td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-mono text-xs">
                  AGENTSMESH_URL
                </td>
                <td className="p-3 border-b border-border">
                  AgentsMesh server URL
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  https://api.agentsmesh.dev
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-mono text-xs">
                  MAX_CONCURRENT_PODS
                </td>
                <td className="p-3 border-b border-border">
                  Maximum concurrent Pods
                </td>
                <td className="p-3 border-b border-border">5</td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-mono text-xs">
                  WORKSPACE_DIR
                </td>
                <td className="p-3 border-b border-border">
                  Base directory for workspaces
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  /data/workspaces
                </td>
              </tr>
              <tr>
                <td className="p-3 font-mono text-xs">MCP_PORT</td>
                <td className="p-3">MCP server port</td>
                <td className="p-3">19000</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Registration Token */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          Creating a Registration Token
        </h2>
        <ol className="list-decimal list-inside text-muted-foreground space-y-2">
          <li>
            Go to <strong>Settings → Runners</strong> in the web interface
          </li>
          <li>
            Click <strong>Create Token</strong>
          </li>
          <li>Set an optional description and expiration</li>
          <li>Copy the generated token</li>
          <li>Use the token when configuring your runner</li>
        </ol>
        <div className="bg-muted rounded-lg p-4 mt-4">
          <p className="text-sm text-muted-foreground">
            <strong>Security Note:</strong> Registration tokens are one-time
            use. Once a runner registers, it receives a permanent auth token.
            You can regenerate the auth token from the runner settings if
            needed.
          </p>
        </div>
      </section>

      {/* Verifying Installation */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Verifying Installation</h2>
        <p className="text-muted-foreground mb-4">
          After starting the runner, verify it&apos;s connected:
        </p>
        <ol className="list-decimal list-inside text-muted-foreground space-y-2">
          <li>
            Go to <strong>Settings → Runners</strong> in the web interface
          </li>
          <li>
            Find your runner in the list - it should show{" "}
            <span className="text-green-400">● Online</span>
          </li>
          <li>
            Try creating an AgentPod using this runner
          </li>
        </ol>
      </section>

      {/* Troubleshooting */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Troubleshooting</h2>
        <div className="space-y-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              Runner shows as Offline
            </h3>
            <p className="text-sm text-muted-foreground">
              Check network connectivity to the AgentsMesh server. Ensure
              firewalls allow outbound gRPC connections (port 9443 for mTLS).
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              Pods fail to start
            </h3>
            <p className="text-sm text-muted-foreground">
              Verify Git is installed and configured. Check that the runner has
              access to clone repositories (SSH keys or tokens configured).
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              &quot;Token invalid&quot; error
            </h3>
            <p className="text-sm text-muted-foreground">
              Registration tokens are single-use. Create a new token from
              Settings → Runners if the original was already used.
            </p>
          </div>
        </div>
      </section>
    </div>
  );
}
