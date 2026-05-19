import type { DemoFrame } from "./types";

const TICKETS = {
  auth: { id: "auth", title: "Auth API", agent: "Claude Code", color: "primary" },
  payment: { id: "payment", title: "Payment Flow", agent: "Codex CLI", color: "blue" },
  tests: { id: "tests", title: "E2E Tests", agent: "Aider", color: "purple" },
  docs: { id: "docs", title: "API Docs", agent: "Gemini CLI", color: "green" },
};

export function getDemoFrames(): DemoFrame[] {
  return [
    {
      time: 0,
      tickets: [
        { ...TICKETS.auth, status: "backlog" },
        { ...TICKETS.payment, status: "backlog" },
        { ...TICKETS.tests, status: "backlog" },
        { ...TICKETS.docs, status: "done" },
      ],
      terminals: [],
      showTerminals: false,
    },
    {
      time: 1500,
      tickets: [
        { ...TICKETS.auth, status: "backlog", executing: true },
        { ...TICKETS.payment, status: "backlog" },
        { ...TICKETS.tests, status: "backlog" },
        { ...TICKETS.docs, status: "done" },
      ],
      terminals: [],
      showTerminals: false,
    },
    {
      time: 2500,
      tickets: [
        { ...TICKETS.auth, status: "in_progress" },
        { ...TICKETS.payment, status: "backlog" },
        { ...TICKETS.tests, status: "backlog" },
        { ...TICKETS.docs, status: "done" },
      ],
      terminals: [
        {
          id: "t1", agent: "Claude Code", ticketId: "auth", active: true,
          lines: [
            { text: "$ claude --pod auth-dev", type: "command" },
            { text: "Analyzing ticket: Auth API...", type: "info" },
          ],
        },
      ],
      showTerminals: true,
    },
    {
      time: 4000,
      tickets: [
        { ...TICKETS.auth, status: "in_progress" },
        { ...TICKETS.payment, status: "backlog", executing: true },
        { ...TICKETS.tests, status: "backlog" },
        { ...TICKETS.docs, status: "done" },
      ],
      terminals: [
        {
          id: "t1", agent: "Claude Code", ticketId: "auth", active: true,
          lines: [
            { text: "$ claude --pod auth-dev", type: "command" },
            { text: "Writing src/auth/handler.ts", type: "output" },
            { text: "Writing src/auth/oauth.ts", type: "output" },
          ],
        },
      ],
      showTerminals: true,
    },
    {
      time: 5000,
      tickets: [
        { ...TICKETS.auth, status: "in_progress" },
        { ...TICKETS.payment, status: "in_progress" },
        { ...TICKETS.tests, status: "backlog" },
        { ...TICKETS.docs, status: "done" },
      ],
      terminals: [
        {
          id: "t1", agent: "Claude Code", ticketId: "auth", active: true,
          lines: [
            { text: "Writing src/auth/handler.ts", type: "output" },
            { text: "Writing src/auth/oauth.ts", type: "output" },
            { text: "Running tests...", type: "info" },
            { text: "✓ 12 tests passed", type: "success" },
          ],
        },
        {
          id: "t2", agent: "Codex CLI", ticketId: "payment", active: true,
          lines: [
            { text: "$ codex --pod payment-dev", type: "command" },
            { text: "Analyzing ticket: Payment Flow...", type: "info" },
          ],
        },
      ],
      showTerminals: true,
    },
    {
      time: 7500,
      tickets: [
        { ...TICKETS.auth, status: "in_progress" },
        { ...TICKETS.payment, status: "in_progress" },
        { ...TICKETS.tests, status: "backlog" },
        { ...TICKETS.docs, status: "done" },
      ],
      terminals: [
        {
          id: "t1", agent: "Claude Code", ticketId: "auth", active: true,
          lines: [
            { text: "Running tests...", type: "info" },
            { text: "✓ 12 tests passed", type: "success" },
            { text: "Creating merge request...", type: "info" },
            { text: "✓ MR !41 created", type: "success" },
          ],
        },
        {
          id: "t2", agent: "Codex CLI", ticketId: "payment", active: true,
          lines: [
            { text: "Writing src/payment/stripe.ts", type: "output" },
            { text: "Writing src/payment/webhook.ts", type: "output" },
            { text: "Running tests...", type: "info" },
          ],
        },
      ],
      showTerminals: true,
    },
    {
      time: 9500,
      tickets: [
        { ...TICKETS.auth, status: "done" },
        { ...TICKETS.payment, status: "in_progress" },
        { ...TICKETS.tests, status: "backlog", executing: true },
        { ...TICKETS.docs, status: "done" },
      ],
      terminals: [
        {
          id: "t2", agent: "Codex CLI", ticketId: "payment", active: true,
          lines: [
            { text: "✓ 8 tests passed", type: "success" },
            { text: "Creating merge request...", type: "info" },
            { text: "✓ MR !42 created", type: "success" },
          ],
        },
      ],
      showTerminals: true,
    },
    {
      time: 10500,
      tickets: [
        { ...TICKETS.auth, status: "done" },
        { ...TICKETS.payment, status: "done" },
        { ...TICKETS.tests, status: "in_progress" },
        { ...TICKETS.docs, status: "done" },
      ],
      terminals: [
        {
          id: "t3", agent: "Aider", ticketId: "tests", active: true,
          lines: [
            { text: "$ aider --pod tests-dev", type: "command" },
            { text: "Analyzing ticket: E2E Tests...", type: "info" },
            { text: "Writing tests/e2e/auth.spec.ts", type: "output" },
          ],
        },
      ],
      showTerminals: true,
    },
    {
      time: 13000,
      tickets: [
        { ...TICKETS.auth, status: "done" },
        { ...TICKETS.payment, status: "done" },
        { ...TICKETS.tests, status: "done" },
        { ...TICKETS.docs, status: "done" },
      ],
      terminals: [],
      showTerminals: false,
    },
  ];
}
