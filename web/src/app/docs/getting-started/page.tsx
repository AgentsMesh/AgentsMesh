import Link from "next/link";

export default function GettingStartedPage() {
  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">Quick Start</h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        Get AgentsMesh up and running in just a few minutes. This guide will walk
        you through the essential setup steps.
      </p>

      {/* Step 1 */}
      <section className="mb-8">
        <div className="border border-border rounded-lg p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-8 h-8 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-sm font-bold">
              1
            </div>
            <h2 className="text-xl font-semibold">Create an Account</h2>
          </div>
          <p className="text-muted-foreground mb-4">
            Sign up at{" "}
            <Link href="/register" className="text-primary hover:underline">
              /register
            </Link>{" "}
            to create your account and organization. You&apos;ll receive a
            verification email to confirm your account.
          </p>
          <div className="bg-muted rounded-lg p-4 text-sm">
            <p className="font-medium mb-2">What you&apos;ll set up:</p>
            <ul className="list-disc list-inside text-muted-foreground space-y-1">
              <li>Personal account with email verification</li>
              <li>Organization (team workspace)</li>
              <li>Initial organization settings</li>
            </ul>
          </div>
        </div>
      </section>

      {/* Step 2 */}
      <section className="mb-8">
        <div className="border border-border rounded-lg p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-8 h-8 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-sm font-bold">
              2
            </div>
            <h2 className="text-xl font-semibold">Configure AI Providers</h2>
          </div>
          <p className="text-muted-foreground mb-4">
            Go to <strong>Settings → AgentPod → AI Providers</strong> to configure
            your AI provider API keys. AgentsMesh supports multiple providers:
          </p>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="bg-muted rounded-lg p-4">
              <h4 className="font-medium mb-2">Anthropic (Claude)</h4>
              <p className="text-sm text-muted-foreground">
                For Claude Code agent. Get your API key from{" "}
                <a
                  href="https://console.anthropic.com"
                  className="text-primary hover:underline"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  console.anthropic.com
                </a>
              </p>
            </div>
            <div className="bg-muted rounded-lg p-4">
              <h4 className="font-medium mb-2">OpenAI</h4>
              <p className="text-sm text-muted-foreground">
                For Codex CLI agent. Get your API key from{" "}
                <a
                  href="https://platform.openai.com"
                  className="text-primary hover:underline"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  platform.openai.com
                </a>
              </p>
            </div>
            <div className="bg-muted rounded-lg p-4">
              <h4 className="font-medium mb-2">Google (Gemini)</h4>
              <p className="text-sm text-muted-foreground">
                For Gemini CLI agent. Get your API key from{" "}
                <a
                  href="https://aistudio.google.com"
                  className="text-primary hover:underline"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  aistudio.google.com
                </a>
              </p>
            </div>
            <div className="bg-muted rounded-lg p-4">
              <h4 className="font-medium mb-2">Custom Provider</h4>
              <p className="text-sm text-muted-foreground">
                For Aider, OpenCode, or custom agents with your own API
                endpoints.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Step 3 */}
      <section className="mb-8">
        <div className="border border-border rounded-lg p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-8 h-8 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-sm font-bold">
              3
            </div>
            <h2 className="text-xl font-semibold">Setup a Runner</h2>
          </div>
          <p className="text-muted-foreground mb-4">
            Runners are the execution environments where AI agent Pods run.
            Download and configure the AgentsMesh Runner on your development
            machine or server.
          </p>
          <div className="bg-[#1a1a1a] rounded-lg p-4 font-mono text-sm overflow-x-auto">
            <pre className="text-green-400">{`# Download and install the runner
curl -fsSL https://get.agentsmesh.dev | sh

# Register with your token (from Settings → Runners)
runner register \\
  --server https://api.agentsmesh.dev \\
  --token <YOUR_TOKEN>

# Start the runner
runner run`}</pre>
          </div>
          <p className="text-sm text-muted-foreground mt-4">
            See{" "}
            <Link
              href="/docs/runners/setup"
              className="text-primary hover:underline"
            >
              Runner Setup
            </Link>{" "}
            for detailed installation instructions and Docker/Kubernetes
            deployment options.
          </p>
        </div>
      </section>

      {/* Step 4 */}
      <section className="mb-8">
        <div className="border border-border rounded-lg p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-8 h-8 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-sm font-bold">
              4
            </div>
            <h2 className="text-xl font-semibold">Start a Pod</h2>
          </div>
          <p className="text-muted-foreground mb-4">
            Create a new AgentPod and start coding with AI assistance!
          </p>
          <ol className="list-decimal list-inside text-muted-foreground space-y-2">
            <li>
              Navigate to <strong>AgentPod</strong> in the sidebar
            </li>
            <li>
              Click <strong>New Pod</strong>
            </li>
            <li>Select an AI agent type (Claude Code, Codex CLI, etc.)</li>
            <li>Choose a Runner to execute the Pod</li>
            <li>Optionally link to a repository and ticket</li>
            <li>
              Click <strong>Create</strong> to start your AI-powered development
              Pod
            </li>
          </ol>
        </div>
      </section>

      {/* Next Steps */}
      <section className="mb-8">
        <h2 className="text-2xl font-semibold mb-4">Next Steps</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Link
            href="/docs/features/agentpod"
            className="border border-border rounded-lg p-4 hover:border-primary transition-colors"
          >
            <h3 className="font-medium mb-1">Learn about AgentPod →</h3>
            <p className="text-sm text-muted-foreground">
              Remote AI development workstations
            </p>
          </Link>
          <Link
            href="/docs/features/mesh"
            className="border border-border rounded-lg p-4 hover:border-primary transition-colors"
          >
            <h3 className="font-medium mb-1">Explore Mesh →</h3>
            <p className="text-sm text-muted-foreground">
              Multi-agent collaboration features
            </p>
          </Link>
        </div>
      </section>
    </div>
  );
}
