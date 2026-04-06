"use client";

interface PromptInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  t: (key: string) => string;
}

/**
 * Initial prompt textarea component
 */
export function PromptInput({
  value,
  onChange,
  placeholder,
  t,
}: PromptInputProps) {
  return (
    <div>
      <label
        htmlFor="prompt-input"
        className="block text-sm font-medium mb-2"
      >
        {t("ide.createPod.prompt")}
      </label>
      <textarea
        id="prompt-input"
        className="w-full px-3 py-2 border border-border rounded-md bg-background resize-none"
        rows={3}
        placeholder={placeholder || t("ide.createPod.promptPlaceholder")}
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
    </div>
  );
}
