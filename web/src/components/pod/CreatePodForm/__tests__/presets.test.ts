import { describe, it, expect } from "vitest";
import {
  ticketPromptGenerator,
  workspacePromptGenerator,
  getScenarioPreset,
  mergeConfig,
} from "../presets";
import { ScenarioContext, CreatePodFormConfig } from "../types";

describe("presets", () => {
  describe("ticketPromptGenerator", () => {
    it("should return empty string when no ticket context", () => {
      const context: ScenarioContext = {};
      expect(ticketPromptGenerator(context)).toBe("");
    });

    it("should generate prompt with ticket slug and title", () => {
      const context: ScenarioContext = {
        ticket: {
          id: 1,
          slug: "PROJ-123",
          title: "Fix login bug",
        },
      };
      const result = ticketPromptGenerator(context);
      expect(result).toBe("Work on ticket PROJ-123: Fix login bug");
    });

    it("should include description when provided", () => {
      const context: ScenarioContext = {
        ticket: {
          id: 1,
          slug: "PROJ-123",
          title: "Fix login bug",
          description: "Users cannot login with valid credentials",
        },
      };
      const result = ticketPromptGenerator(context);
      expect(result).toContain("Work on ticket PROJ-123: Fix login bug");
      expect(result).toContain("Ticket Description:");
      expect(result).toContain("Users cannot login with valid credentials");
    });

    it("should truncate long descriptions to 500 characters", () => {
      const longDescription = "A".repeat(600);
      const context: ScenarioContext = {
        ticket: {
          id: 1,
          slug: "PROJ-123",
          title: "Fix bug",
          description: longDescription,
        },
      };
      const result = ticketPromptGenerator(context);
      expect(result).toContain("A".repeat(500) + "...");
      expect(result).not.toContain("A".repeat(501));
    });

    it("should not truncate descriptions under 500 characters", () => {
      const shortDescription = "A".repeat(400);
      const context: ScenarioContext = {
        ticket: {
          id: 1,
          slug: "PROJ-123",
          title: "Fix bug",
          description: shortDescription,
        },
      };
      const result = ticketPromptGenerator(context);
      expect(result).toContain(shortDescription);
      expect(result).not.toContain("...");
    });

    it("should handle exactly 500 character descriptions", () => {
      const exactDescription = "A".repeat(500);
      const context: ScenarioContext = {
        ticket: {
          id: 1,
          slug: "PROJ-123",
          title: "Fix bug",
          description: exactDescription,
        },
      };
      const result = ticketPromptGenerator(context);
      expect(result).toContain(exactDescription);
      expect(result).not.toContain("...");
    });
  });

  describe("workspacePromptGenerator", () => {
    it("should always return empty string", () => {
      expect(workspacePromptGenerator({})).toBe("");
      expect(workspacePromptGenerator({ ticket: { id: 1, slug: "X", title: "Y" } })).toBe("");
    });
  });

  describe("getScenarioPreset", () => {
    it("should return ticket preset for ticket scenario", () => {
      const preset = getScenarioPreset("ticket");
      expect(preset.scenario).toBe("ticket");
      expect(preset.promptGenerator).toBe(ticketPromptGenerator);
    });

    it("should return workspace preset for workspace scenario", () => {
      const preset = getScenarioPreset("workspace");
      expect(preset.scenario).toBe("workspace");
      expect(preset.promptGenerator).toBe(workspacePromptGenerator);
    });

    it("should default to workspace preset for unknown scenario", () => {
      // @ts-expect-error - testing invalid scenario
      const preset = getScenarioPreset("unknown");
      expect(preset.scenario).toBe("workspace");
      expect(preset.promptGenerator).toBe(workspacePromptGenerator);
    });
  });

  describe("mergeConfig", () => {
    it("should merge user config with preset", () => {
      const config: CreatePodFormConfig = {
        scenario: "ticket",
        onSuccess: () => {},
      };
      const merged = mergeConfig(config);
      expect(merged.scenario).toBe("ticket");
      expect(merged.promptGenerator).toBe(ticketPromptGenerator);
      expect(merged.onSuccess).toBe(config.onSuccess);
    });

    it("should preserve user-provided promptGenerator", () => {
      const customGenerator = () => "custom prompt";
      const config: CreatePodFormConfig = {
        scenario: "ticket",
        promptGenerator: customGenerator,
      };
      const merged = mergeConfig(config);
      expect(merged.promptGenerator).toBe(customGenerator);
    });

    it("should use preset promptGenerator when user does not provide one", () => {
      const config: CreatePodFormConfig = {
        scenario: "workspace",
      };
      const merged = mergeConfig(config);
      expect(merged.promptGenerator).toBe(workspacePromptGenerator);
    });

    it("should preserve all user config properties", () => {
      const onSuccess = () => {};
      const onError = () => {};
      const onCancel = () => {};
      const config: CreatePodFormConfig = {
        scenario: "ticket",
        context: { ticket: { id: 1, slug: "X", title: "Y" } },
        promptPlaceholder: "Custom placeholder",
        onSuccess,
        onError,
        onCancel,
      };
      const merged = mergeConfig(config);
      expect(merged.context).toBe(config.context);
      expect(merged.promptPlaceholder).toBe("Custom placeholder");
      expect(merged.onSuccess).toBe(onSuccess);
      expect(merged.onError).toBe(onError);
      expect(merged.onCancel).toBe(onCancel);
    });
  });
});
