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

    it("should generate prompt with ticket identifier and title", () => {
      const context: ScenarioContext = {
        ticket: {
          id: 1,
          identifier: "PROJ-123",
          title: "Fix login bug",
        },
      };
      const result = ticketPromptGenerator(context);
      expect(result).toBe("Work on ticket PROJ-123: Fix login bug");
    });
  });

  describe("workspacePromptGenerator", () => {
    it("should always return empty string", () => {
      expect(workspacePromptGenerator({})).toBe("");
      expect(workspacePromptGenerator({ ticket: { id: 1, identifier: "X", title: "Y" } })).toBe("");
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
        context: { ticket: { id: 1, identifier: "X", title: "Y" } },
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
