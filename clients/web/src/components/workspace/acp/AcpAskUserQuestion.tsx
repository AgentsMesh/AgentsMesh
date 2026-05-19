"use client";

import { useState } from "react";
import { HelpCircle } from "lucide-react";
import type { AcpPermissionRequest } from "@/stores/acpSession";

interface AskUserQuestionOption {
  label: string;
  description: string;
}

interface AskUserQuestion {
  question: string;
  header: string;
  options: AskUserQuestionOption[];
  multiSelect: boolean;
}

interface AcpAskUserQuestionProps {
  permission: AcpPermissionRequest;
  onRespond: (requestId: string, approved: boolean, updatedInput?: Record<string, unknown>) => void;
}

export function AcpAskUserQuestion({ permission, onRespond }: AcpAskUserQuestionProps) {
  const parsed = parseQuestions(permission.argumentsJson);
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const [multiAnswers, setMultiAnswers] = useState<Record<string, string[]>>({});
  const [customInputs, setCustomInputs] = useState<Record<string, string>>({});

  if (!parsed || parsed.length === 0) {
    return <FallbackView permission={permission} onRespond={onRespond} />;
  }

  const handleSelect = (question: string, label: string) => {
    setAnswers((prev) => ({ ...prev, [question]: label }));
  };

  const handleMultiToggle = (question: string, label: string) => {
    setMultiAnswers((prev) => {
      const current = prev[question] || [];
      const idx = current.indexOf(label);
      if (idx >= 0) {
        return { ...prev, [question]: current.filter((_, i) => i !== idx) };
      }
      return { ...prev, [question]: [...current, label] };
    });
  };

  const handleSubmit = () => {
    const finalAnswers: Record<string, string> = {};
    for (const q of parsed) {
      const custom = customInputs[q.question];
      if (custom) {
        finalAnswers[q.question] = custom;
      } else if (q.multiSelect) {
        finalAnswers[q.question] = (multiAnswers[q.question] || []).join(", ");
      } else {
        finalAnswers[q.question] = answers[q.question] || "";
      }
    }
    onRespond(permission.requestId, true, {
      questions: parsed,
      answers: finalAnswers,
    });
  };

  const handleSkip = () => {
    onRespond(permission.requestId, false);
  };

  return (
    <div className="rounded-lg border border-blue-200 dark:border-blue-800 bg-blue-50 dark:bg-blue-950/30 p-3 space-y-3">
      {parsed.map((q) => (
        <QuestionBlock
          key={q.question}
          question={q}
          selectedLabels={q.multiSelect ? (multiAnswers[q.question] || []) : (answers[q.question] ? [answers[q.question]] : [])}
          customInput={customInputs[q.question] || ""}
          onSelect={q.multiSelect ? handleMultiToggle : handleSelect}
          onCustomInput={(val) => setCustomInputs((prev) => ({ ...prev, [q.question]: val }))}
        />
      ))}
      <div className="flex gap-2 pt-1">
        <button
          onClick={handleSubmit}
          className="rounded bg-blue-600 px-3 py-1 text-xs text-white hover:bg-blue-700"
        >
          Submit
        </button>
        <button
          onClick={handleSkip}
          className="rounded bg-muted px-3 py-1 text-xs hover:bg-muted/80"
        >
          Skip
        </button>
      </div>
    </div>
  );
}

function QuestionBlock({
  question,
  selectedLabels,
  customInput,
  onSelect,
  onCustomInput,
}: {
  question: AskUserQuestion;
  selectedLabels: string[];
  customInput: string;
  onSelect: (question: string, label: string) => void;
  onCustomInput: (val: string) => void;
}) {

  return (
    <div>
      <div className="flex items-center gap-1.5 mb-1.5">
        <HelpCircle className="h-3.5 w-3.5 text-blue-500" />
        <span className="text-sm font-medium">{question.question}</span>
      </div>
      <div className="space-y-1 ml-5">
        {question.options.map((opt) => {
          const isSelected = selectedLabels.includes(opt.label) && !customInput;
          return (
            <button
              key={opt.label}
              onClick={() => {
                onCustomInput("");
                onSelect(question.question, opt.label);
              }}
              className={`block w-full text-left rounded px-2 py-1 text-xs transition-colors ${
                isSelected
                  ? "bg-blue-100 dark:bg-blue-900 border border-blue-300 dark:border-blue-700"
                  : "bg-muted/50 hover:bg-muted border border-transparent"
              }`}
            >
              <span className="font-medium">{opt.label}</span>
              {opt.description && (
                <span className="text-muted-foreground ml-1">— {opt.description}</span>
              )}
            </button>
          );
        })}
        <input
          type="text"
          value={customInput}
          onChange={(e) => onCustomInput(e.target.value)}
          placeholder="Other..."
          className="w-full rounded border bg-background px-2 py-1 text-xs"
        />
      </div>
    </div>
  );
}

/** Fallback when argumentsJson can't be parsed as questions. */
function FallbackView({
  permission,
  onRespond,
}: {
  permission: AcpPermissionRequest;
  onRespond: (requestId: string, approved: boolean) => void;
}) {
  return (
    <div className="rounded-lg border border-blue-200 dark:border-blue-800 p-3">
      <div className="flex items-center gap-2 mb-2">
        <HelpCircle className="h-4 w-4 text-blue-500" />
        <span className="text-sm font-medium">Question</span>
      </div>
      <p className="text-sm mb-2">{permission.description}</p>
      <div className="flex gap-2">
        <button
          onClick={() => onRespond(permission.requestId, true)}
          className="rounded bg-blue-600 px-3 py-1 text-xs text-white hover:bg-blue-700"
        >
          OK
        </button>
        <button
          onClick={() => onRespond(permission.requestId, false)}
          className="rounded bg-muted px-3 py-1 text-xs hover:bg-muted/80"
        >
          Skip
        </button>
      </div>
    </div>
  );
}

function parseQuestions(json: string): AskUserQuestion[] | null {
  try {
    const data = JSON.parse(json);
    if (data?.questions && Array.isArray(data.questions)) {
      return data.questions;
    }
  } catch {
  }
  return null;
}
