"use client";

import { useState, useRef, useMemo, useCallback } from "react";
import type { KeyboardEvent } from "react";
import { useTranslations } from "next-intl";
import { Send, StopCircle } from "lucide-react";
import { relayPool } from "@/stores/relayConnection";
import { useAcpSessionField } from "@/stores/acpSession";
import { useLoopalSession } from "@/stores/loopalConsole";
import { loopalControl } from "../loopalControl";
import { buildLoopalCommands, parseSlashInput, filterCommands, type SlashCommand } from "./loopalCommands";
import { getSlashQuery } from "./loopalSlashQuery";
import { LoopalSlashDropdown } from "./LoopalSlashDropdown";
import { LoopalModelThinkingPopover } from "./LoopalModelThinkingPopover";
import { LoopalInputPopover } from "./LoopalInputPopover";

type Popover = "model" | "rewind" | "resume" | null;

export function LoopalPromptInput({ podKey }: { podKey: string }) {
  const t = useTranslations("loopal");
  const [text, setText] = useState("");
  const [visible, setVisible] = useState(false);
  const [active, setActive] = useState(0);
  const [popover, setPopover] = useState<Popover>(null);
  const taRef = useRef<HTMLTextAreaElement>(null);

  const sessionState = useAcpSessionField(podKey, (s) => s.state);
  const { thread_goal, model, thinking } = useLoopalSession(podKey);
  const isProcessing = sessionState === "processing" || sessionState === "waiting_permission";

  const commands = useMemo(() => buildLoopalCommands({ hasGoal: !!thread_goal, t }), [thread_goal, t]);
  const query = getSlashQuery(text, text.length)?.query ?? "";
  const matches = useMemo(() => (visible ? filterCommands(commands, query) : []), [visible, commands, query]);
  const safeActive = Math.min(active, Math.max(matches.length - 1, 0));

  const focusEnd = useCallback(() => {
    requestAnimationFrame(() => {
      const ta = taRef.current;
      if (!ta) return;
      ta.focus();
      const p = ta.value.length;
      ta.setSelectionRange(p, p);
    });
  }, []);

  function onChange(v: string) {
    setText(v);
    setVisible(!!getSlashQuery(v, v.length));
    setActive(0);
  }

  const execute = useCallback(
    (command: SlashCommand, arg: string) => {
      setVisible(false);
      const a = command.action;
      if (a.kind === "popover") {
        setPopover(a.popover);
        return;
      }
      if (a.kind === "goal-create") {
        if (arg) {
          loopalControl(podKey, "loopal.goalCreate", { objective: arg });
          setText("");
        } else {
          setText(command.label + " ");
          focusEnd();
        }
        return;
      }
      loopalControl(podKey, a.subtype, a.payload);
      setText("");
    },
    [podKey, focusEnd],
  );

  function send() {
    const trimmed = text.trim();
    if (!trimmed) return;
    const parsed = parseSlashInput(trimmed, commands);
    if (parsed) {
      execute(parsed.command, parsed.arg);
      return;
    }
    // Slash controls are always allowed; a plain prompt is gated like the
    // workspace composer — no send while the agent is mid-turn (the Send button
    // is swapped for Interrupt, and Enter must honor the same affordance).
    if (isProcessing) return;
    if (!relayPool.isConnected(podKey)) return;
    relayPool.sendAcpCommand(podKey, { type: "prompt", prompt: text });
    setText("");
  }

  function onKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
    if (e.nativeEvent.isComposing) return;
    if (visible && matches.length > 0) {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setActive((p) => (p < matches.length - 1 ? p + 1 : 0));
        return;
      }
      if (e.key === "ArrowUp") {
        e.preventDefault();
        setActive((p) => (p > 0 ? p - 1 : matches.length - 1));
        return;
      }
      if (e.key === "Enter" || e.key === "Tab") {
        e.preventDefault();
        execute(matches[safeActive], "");
        return;
      }
      if (e.key === "Escape") {
        e.preventDefault();
        setVisible(false);
        return;
      }
    }
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      send();
    }
  }

  function closePopover() {
    setPopover(null);
    focusEnd();
  }

  return (
    <div className="relative border-t border-border px-3 py-2">
      <LoopalSlashDropdown commands={matches} activeIndex={safeActive} onSelect={(c) => execute(c, "")} visible={visible} />
      {popover === "model" && (
        <LoopalModelThinkingPopover
          currentModel={model}
          currentThinking={thinking}
          onPick={(config) => {
            loopalControl(podKey, "loopal.thinking", { config });
            closePopover();
          }}
          onClose={closePopover}
        />
      )}
      {popover === "rewind" && (
        <LoopalInputPopover
          title={t("prompt.rewindTitle")}
          placeholder={t("prompt.rewindPlaceholder")}
          numeric
          onSubmit={(v) => {
            loopalControl(podKey, "loopal.rewind", { turn_index: Number(v) });
            closePopover();
          }}
          onClose={closePopover}
        />
      )}
      {popover === "resume" && (
        <LoopalInputPopover
          title={t("prompt.resumeTitle")}
          placeholder={t("prompt.resumePlaceholder")}
          onSubmit={(v) => {
            loopalControl(podKey, "loopal.resumeSession", { session_id: v });
            closePopover();
          }}
          onClose={closePopover}
        />
      )}
      <div className="flex items-end gap-2">
        <textarea
          ref={taRef}
          value={text}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={onKeyDown}
          placeholder={t("prompt.placeholder")}
          className="min-h-[36px] max-h-[120px] flex-1 resize-none rounded-md border border-border bg-background px-3 py-1.5 text-sm leading-[20px] outline-none focus:border-ring"
          rows={1}
          data-testid="loopal-prompt-input"
        />
        {isProcessing ? (
          <button
            type="button"
            onClick={() => relayPool.isConnected(podKey) && relayPool.sendAcpCommand(podKey, { type: "interrupt" })}
            title={t("prompt.interrupt")}
            className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-red-600 text-white hover:bg-red-700"
          >
            <StopCircle className="h-4 w-4" />
          </button>
        ) : (
          <button
            type="button"
            onClick={send}
            disabled={!text.trim()}
            className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            <Send className="h-4 w-4" />
          </button>
        )}
      </div>
    </div>
  );
}
