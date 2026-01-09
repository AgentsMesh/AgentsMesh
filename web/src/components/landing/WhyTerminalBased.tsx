"use client";

const benefits = [
  {
    icon: (
      <svg className="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M13 10V3L4 14h7v7l9-11h-7z" />
      </svg>
    ),
    title: "Autonomous Execution",
    description: "Not just code completion. Terminal agents can independently complete complex development tasks from start to finish.",
  },
  {
    icon: (
      <svg className="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
      </svg>
    ),
    title: "Full Capabilities",
    description: "Read/write files, execute commands, Git operations, run tests - complete full-stack development capabilities.",
  },
  {
    icon: (
      <svg className="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
      </svg>
    ),
    title: "Data Control",
    description: "Self-hosted runners mean your code never leaves your infrastructure. Meet security and compliance requirements.",
  },
];

const comparisonData = [
  { feature: "Autonomy", ide: "Passive response", terminal: "Active task execution", highlight: true },
  { feature: "Capabilities", ide: "Code completion", terminal: "Full-stack development", highlight: true },
  { feature: "Environment", ide: "IDE sandbox", terminal: "Full terminal access", highlight: true },
  { feature: "Multi-Agent", ide: "Not supported", terminal: "AgentMesh enabled", highlight: true },
  { feature: "Self-hosted", ide: "Cloud only", terminal: "Fully controllable", highlight: true },
];

export function WhyTerminalBased() {
  return (
    <section className="py-24 relative" id="why-terminal">
      {/* Background */}
      <div className="absolute inset-0 bg-gradient-to-b from-transparent via-primary/5 to-transparent" />

      <div className="container mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        {/* Section header */}
        <div className="text-center mb-16">
          <h2 className="text-3xl sm:text-4xl font-bold mb-4">
            Why <span className="text-primary">Terminal-based</span> Agent?
          </h2>
          <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
            Unlike IDE plugins like Cursor or Copilot, terminal-based agents offer
            autonomous execution and complete development capabilities.
          </p>
        </div>

        {/* Benefits cards */}
        <div className="grid md:grid-cols-3 gap-6 mb-16">
          {benefits.map((benefit, index) => (
            <div
              key={index}
              className="p-6 bg-secondary/20 rounded-xl border border-border hover:border-primary/50 transition-all duration-300 hover:-translate-y-1"
            >
              <div className="w-14 h-14 rounded-lg bg-primary/10 flex items-center justify-center text-primary mb-4">
                {benefit.icon}
              </div>
              <h3 className="text-xl font-semibold mb-2">{benefit.title}</h3>
              <p className="text-muted-foreground">{benefit.description}</p>
            </div>
          ))}
        </div>

        {/* Comparison table */}
        <div className="max-w-4xl mx-auto">
          <div className="bg-[#0d0d0d] rounded-xl border border-border overflow-hidden">
            {/* Table header */}
            <div className="grid grid-cols-3 bg-[#1a1a1a] border-b border-border">
              <div className="p-4 font-semibold text-sm"></div>
              <div className="p-4 font-semibold text-sm text-center text-muted-foreground">
                IDE Plugins
                <div className="text-xs font-normal mt-1">Copilot / Cursor</div>
              </div>
              <div className="p-4 font-semibold text-sm text-center text-primary">
                Terminal-based ✓
                <div className="text-xs font-normal mt-1 text-primary/70">Claude Code / Aider</div>
              </div>
            </div>

            {/* Table rows */}
            {comparisonData.map((row, index) => (
              <div
                key={index}
                className={`grid grid-cols-3 border-b border-border last:border-b-0 ${
                  index % 2 === 0 ? "bg-transparent" : "bg-secondary/10"
                }`}
              >
                <div className="p-4 font-medium text-sm">{row.feature}</div>
                <div className="p-4 text-sm text-center text-muted-foreground">
                  <span className="inline-flex items-center gap-1">
                    <svg className="w-4 h-4 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                    {row.ide}
                  </span>
                </div>
                <div className="p-4 text-sm text-center">
                  <span className="inline-flex items-center gap-1 text-green-400">
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    {row.terminal}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
