---
title: "Building AgentsMesh with AgentsMesh: 52 Days of Harness Engineering by One Person"
excerpt: "OpenAI calls it Harness Engineering. Using this methodology, one person in 52 days, 600 commits, and 965,687 lines of code throughput, built the Harness Engineering tool itself. The codebase is the context. The engineering environment sets the ceiling for agent output quality."
date: "2026-03-04"
author: "AgentsMesh Team"
category: "Insight"
readTime: 12
---

OpenAI recently published a piece describing how they used AI agents to produce over a million lines of code in five months. They call this engineering practice **Harness Engineering**.

I started building **AgentsMesh** a little over 50 days ago. 52 days, 600 commits, 965,687 lines of code throughput, 356,220 lines of production code still standing. One person.

But what's worth talking about isn't the numbers — it's the structure of the endeavor itself: I used Harness Engineering to build a Harness Engineering tool.

The repository is fully open source and the git history is public. Every number here is verifiable via git log.

## The Engineering Environment Sets the Ceiling for Agent Output

52 days of hands-on work convinced me that agent output quality depends not just on the agent itself, but critically on the engineering soil it works in. These are real choices that have sedimented into the codebase.

### Layered Architecture Tells the Agent Where to Make Changes

The codebase follows strict **DDD** layering: the domain layer contains only data structures, the service layer only business logic, the handler layer only HTTP format conversion. 22 domain modules with clear boundaries, each module's interface.go explicitly defining its public contract.

When an agent needs to add a new feature, it knows: data structures go in domain, business rules go in service, routes go in handler. In a codebase with blurry boundaries, the agent puts things in the wrong place; in a codebase with clear boundaries, agent-generated code integrates naturally. This isn't theoretical architectural cleanliness — it's the navigation map the agent uses when generating code.

### Directory Structure as Documentation

Cross-stack naming is fully aligned. Take Loop as an example: backend/internal/domain/loop/ for data structures, backend/internal/service/loop/ for business logic, web/src/components/loops/ for frontend components. The mapping from product concept to code path is direct — no searching needed, the directory name is the map.

The backend's 16 domain modules (agentpod, channel, ticket, loop, runner...) mirror the service layer 1:1; the web's components are organized by product function (pod, tickets, loops, mesh, workspace), aligned with backend domain naming. An agent receiving a Ticket-related task doesn't need to explore the entire codebase — the directory structure alone tells it where to work.

This convention wasn't written in a document. It was continuously reinforced by every agent commit across the entire codebase.

### Tech Debt Gets Amplified Exponentially by Agents

This was one of the most counterintuitive discoveries of the 52 days.

When you make a temporary compromise in a module — bypass the service layer to query the database directly, or use a hardcoded magic number — the agent picks up that pattern. Next time it generates code for similar functionality, it reuses this "precedent." Not occasionally, but systematically. Tech debt is no longer isolated; it starts spreading.

A human engineer encountering bad code usually knows "this is a landmine, step around it." An agent doesn't make that judgment — it sees: this pattern exists in the codebase, so it's a legitimate approach.

This means code quality signals in the repository matter far more than when humans write code alone. When good engineering practices dominate, the agent amplifies good practices; when temporary compromises dominate, the agent amplifies tech debt.

The practical response: I stopped multiple times mid-stream specifically to clean up tech debt — no new features, only refactoring. Not to make the code "look nice," but to maintain the purity of engineering signals in the repository. This is a maintenance cost unique to agent-collaborative development, and one of the biggest differences from traditional development rhythms.

### Strong Typing as a Compile-Time Quality Gate

Go + TypeScript + Proto. Strong typing shifts a massive number of errors from runtime to compile time.

Agent generates a function with a mismatched signature? Build failure. Agent modifies an API format but forgets to update the type definition? TypeScript catches it immediately. Agent changes the Runner's message format without syncing the Backend? The Proto-generated code won't compile.

These errors would silently slip into runtime in weakly typed languages. Strong typing blocks them before commit. The shorter the feedback loop, the higher the agent's iteration efficiency.

### Four-Layer Feedback Loop

An agent needs to know quickly what it did wrong. One layer isn't enough; four is just right. And the shorter and more precise the feedback loop, the better the agent's deliverables.

Layer one: compilation. Air hot-reload — Go code restarts within 1 second of modification; TypeScript type errors are flagged in real time. Syntax and type-level errors are eliminated at this layer.

Layer two: unit tests. 700+ tests covering the domain and service layers. The agent knows within 5 minutes whether it introduced a regression, especially for boundary conditions like multi-tenant isolation that agents tend to miss.

Layer three: e2e tests. End-to-end validation of real functional paths. Covers integration boundaries that unit tests can't reach — the actual interplay between multiple modules.

Layer four: CI pipeline. Every PR automatically runs the full test suite, linting, type-checking, and multi-platform build verification. The last safety net before merge, executed by machines, independent of the reviewer's attentiveness.

The four layers have increasing latency and expanding error coverage. When an agent changes one line of code, layer one confirms; when an agent does cross-module refactoring, only layer four can fully verify.

### Worktrees Enable Native Parallelism

dev.sh automatically calculates port offsets based on the Git worktree name, assigning each worktree an independent port range. Multiple agents work simultaneously in different worktrees with fully isolated environments — no manual port conflict management needed.

This is the Pod isolation primitive extended to the development environment level — the same logic, carried from the agent execution environment all the way to the agent development environment.

### The Codebase Is the Agent's Context, Not Just the Prompt

Put all of the above together, and they point to the same conclusion: the codebase itself is the most important context when the agent works. Layered architecture tells the agent where to make changes; directory structure tells the agent which file to find; the cleanliness of tech debt determines whether the agent learns good patterns or bad ones; test density determines how boldly the agent can refactor; strong typing determines how early the agent's errors get caught.

This means you don't need to build a context system outside the codebase — no need for deliberate Context Engineering, no need for a separate RAG setup, no need to maintain extra memory files. What you need to do is make the codebase itself a high-quality context. **The repository is the context.**

**This is also why the investment direction for Harness Engineering is the same thing as traditional software engineering**: writing clear code, maintaining good architecture, cleaning up tech debt promptly. The only difference is the purpose — before, it was to make things easier for human engineers to maintain; now, it's also to let AI agents work reliably.

## Cognitive Bandwidth Is a Real Engineering Constraint

Around day 5, I hit a real wall — roughly 50,000 lines of daily code throughput.

Three worktrees open simultaneously, three agents running, and I was switching between them making decisions. Add a fourth, and decision quality dropped noticeably. Not a feeling — I later discovered that period had left behind several bad architectural decisions.

The daily throughput ceiling of 50,000 lines isn't a tooling limitation; it's the natural upper bound of human cognitive bandwidth. You can make truly sound architectural decisions for about three parallel workstreams. Beyond that, quality starts slipping.

The only way to break through: trade delegation for scale. Not giving the agent more tasks, but delegating the decision-making itself. Let agents coordinate agents, and move yourself up one level — from supervising individual agents to supervising the system that supervises agents. That's how **Autopilot** mode was born.

This is the core design intent of AgentsMesh. And something I only truly understood through the process of using it to build itself.

## When Experimentation Costs Collapse, Engineering Methodology Must Evolve

The Relay architecture in AgentsMesh wasn't designed. It was forged in production.

Three Pods running simultaneously hammered the Backend down. I watched it crash, understood why, and rebuilt. Added Relay to isolate terminal traffic. New problems emerged — added intelligent aggregation, added on-demand connection management. The final architecture came from a succession of real failures, not whiteboard discussions.

The old engineering instinct says design first, then build — thoroughly reason through edge cases, because mistakes are expensive.

When experimentation costs approach zero, that instinct becomes a liability.

That Relay failure went from discovery to fix in under two days. In a traditional team, that's two weeks of architecture discussion — and the discussion would inevitably miss something.

**AI doesn't change the speed of writing code; it changes the cost structure of the entire engineering process.** When iteration is cheap enough, experiment-driven development produces better architectures than design-driven development — and faster. The standard for architectural correctness isn't passing a review; it's surviving production.

## Self-Bootstrapping as Validation

The core claim of AgentsMesh: AI agents can collaboratively complete complex engineering tasks under a structured Harness.

I used AgentsMesh to build AgentsMesh.

This is the most direct test of that claim. If Harness Engineering truly works, this tool should be capable of building itself.

52 days, 965,687 lines of code throughput, 356,220 lines of production code, 600 commits, one author.

OpenAI was a team, working over five months. This isn't a comparison — different scenarios, different scales. But one thing is the same: the Harness makes output possible that would otherwise be unthinkable.

The commit history is the evidence. Any engineer can clone the repository, run git log --numstat — the numbers don't change depending on who's looking.

## Three Engineering Primitives

52 days of practice and self-bootstrapping validation ultimately converged into three engineering primitives. This isn't a pre-designed product framework — it was forged by real engineering problems.

**Isolation**
Every agent needs its own independent workspace. Not a best practice — a hard prerequisite. Without isolation, parallel work is structurally impossible. AgentsMesh implements this with **Pods**: each agent runs in its own Git worktree and sandbox. Conflicts go from "could happen" to "structurally cannot happen." Isolation also means cohesion — within the isolated Pod environment, all the context the agent needs for execution is prepared: Repo, Skills, MCP and more. In practice, building a Pod is the process of preparing the execution environment for the agent.

**Decomposition**
Agents aren't good at "help me with this codebase." They're good at: you own this scope, these are the acceptance criteria, this is the definition of done. Ownership isn't just task assignment — it changes how the agent reasons. Decomposition is the engineering work that must be completed before any agent runs.

AgentsMesh provides two abstractions for decomposition: **Tickets** map to one-shot work items ��� feature development, bug fixes, refactoring, with full kanban status flow and MR association; **Loops** map to recurring automated tasks — daily tests, scheduled builds, code quality scans, scheduled via Cron expressions, each run leaving an independent LoopRun record. The boundary between the two task forms is clear: do something once, use a Ticket; do the same thing repeatedly, use a Loop.

**Coordination**
We don't use role-based abstractions to organize agent collaboration. Traditional teams need job roles because each person only masters a few specialties — frontend engineers don't write backend code, product managers don't write code. But agents aren't bound by this constraint: the same agent can write code, generate documentation, do competitive analysis, run tests, review PRs, and even orchestrate other agents' workflows. Its capability boundaries aren't fixed — they're configured through context and tools. So coordination between agents doesn't need to mimic human division of labor; it needs communication and permissions.

**Channels** address the collective level: multiple Pods sharing messages, decisions, and documents in the same collaboration space. This is the foundation for Supervisor Agents and Worker Agents to form collaborative structures — not a group chat, but a structured communication layer with contextual history.

**Bindings** address the capability level: point-to-point permission grants between two Pods. **terminal:read** lets one agent observe another agent's terminal output; **terminal:write** lets one agent directly control another agent's execution. Bindings are the mechanism for agents coordinating agents — the Supervisor doesn't perceive the Worker's state by sending messages, but by directly seeing its terminal.

OpenAI calls the corresponding concepts Context Engineering, architectural constraints, and entropy management. Different names, same problem.

## Open Source

Harness Engineering is an engineering discipline, not a product feature. Rather than keeping it to ourselves, we'd rather put it out there to inspire what comes next.

We chose to open source AgentsMesh. When what we've built might be an effective engineering tool, the goal was never "owning the code" — it was enabling more people to build even better engineering tools on this foundation. Rather than locking potentially sound engineering practices inside a product, we open source them so the community can verify, evolve, and surpass them.

The code is on [GitHub](https://github.com/AgentsMesh/AgentsMesh)

You can use it to: deploy your own Runner and run AI agents in local isolated environments; manage agent workflows with Tickets and Loops; coordinate multiple agents on complex tasks through Channels and Bindings.

If you've made your own discoveries while practicing Harness Engineering — come chat on [GitHub Discussions](https://github.com/AgentsMesh/AgentsMesh/discussions), or open an [Issue](https://github.com/AgentsMesh/AgentsMesh/issues) directly. This project was built with agents. It should continue to evolve through agents and engineers together.
