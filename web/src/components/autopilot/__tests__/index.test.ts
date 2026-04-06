import { describe, it, expect } from "vitest";
// Import directly from index.ts to ensure coverage
import * as panelExports from "../panel/index";

describe("panel/index exports", () => {
  it("should export ThinkingTab component", () => {
    expect(panelExports.ThinkingTab).toBeDefined();
    expect(typeof panelExports.ThinkingTab).toBe("function");
  });

  it("should export ProgressTab component", () => {
    expect(panelExports.ProgressTab).toBeDefined();
    expect(typeof panelExports.ProgressTab).toBe("function");
  });

  it("should export HistoryTab component", () => {
    expect(panelExports.HistoryTab).toBeDefined();
    expect(typeof panelExports.HistoryTab).toBe("function");
  });

  it("should export ControlButtons component", () => {
    expect(panelExports.ControlButtons).toBeDefined();
    expect(typeof panelExports.ControlButtons).toBe("function");
  });

  it("should export normalizeDecisionType function", () => {
    expect(panelExports.normalizeDecisionType).toBeDefined();
    expect(typeof panelExports.normalizeDecisionType).toBe("function");
  });

  it("should export decisionConfig", () => {
    expect(panelExports.decisionConfig).toBeDefined();
    expect(panelExports.decisionConfig.continue).toBeDefined();
    expect(panelExports.decisionConfig.completed).toBeDefined();
    expect(panelExports.decisionConfig.need_help).toBeDefined();
    expect(panelExports.decisionConfig.give_up).toBeDefined();
  });

  it("should export actionConfig", () => {
    expect(panelExports.actionConfig).toBeDefined();
    expect(panelExports.actionConfig.observe).toBeDefined();
    expect(panelExports.actionConfig.send_input).toBeDefined();
    expect(panelExports.actionConfig.wait).toBeDefined();
    expect(panelExports.actionConfig.none).toBeDefined();
  });

  it("should export iterationPhaseConfig", () => {
    expect(panelExports.iterationPhaseConfig).toBeDefined();
    expect(panelExports.iterationPhaseConfig.prompt).toBeDefined();
    expect(panelExports.iterationPhaseConfig.control_running).toBeDefined();
    expect(panelExports.iterationPhaseConfig.completed).toBeDefined();
  });

  it("should export phaseConfig", () => {
    expect(panelExports.phaseConfig).toBeDefined();
    expect(panelExports.phaseConfig.running).toBeDefined();
    expect(panelExports.phaseConfig.paused).toBeDefined();
    expect(panelExports.phaseConfig.completed).toBeDefined();
  });
});
