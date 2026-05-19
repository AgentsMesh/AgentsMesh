"use client";

import React from "react";

import type { ColumnSpec, JSONMap, SelectOption } from "@/lib/api/blockstoreTypes";
import { cn } from "@/lib/utils";
import { useBlock } from "@/stores/blockstore";

import { UserPicker } from "./UserPicker";

export interface FieldRendererProps {
  column: ColumnSpec;
  value: unknown;
  onChange: (next: unknown) => void;
  readOnly?: boolean;
}

export function FieldRenderer({ column, value, onChange, readOnly }: FieldRendererProps) {
  switch (column.type) {
    case "text":
      return <TextField value={asString(value)} onChange={onChange} readOnly={readOnly} placeholder={column.label ?? column.key} />;
    case "number":
      return <NumberField value={asNumber(value)} onChange={onChange} readOnly={readOnly} />;
    case "boolean":
      return <BooleanField value={asBoolean(value)} onChange={onChange} readOnly={readOnly} />;
    case "select":
      return <SelectField options={column.options ?? []} value={asString(value)} onChange={onChange} readOnly={readOnly} />;
    case "multi_select":
      return <MultiSelectField options={column.options ?? []} value={asStringArray(value)} onChange={onChange} readOnly={readOnly} />;
    case "date":
      return <DateField value={asString(value)} onChange={onChange} readOnly={readOnly} />;
    case "url":
      return <URLField value={asString(value)} onChange={onChange} readOnly={readOnly} />;
    case "user":
      return <UserField value={asNumber(value)} onChange={onChange} readOnly={readOnly} />;
    case "block_ref":
      return <BlockRefField value={asString(value)} onChange={onChange} readOnly={readOnly} />;
    default:
      return <span className="text-xs text-muted-foreground">Unsupported type: {column.type}</span>;
  }
}

function TextField({ value, onChange, readOnly, placeholder }: {
  value: string; onChange: (v: string) => void; readOnly?: boolean; placeholder?: string;
}) {
  return (
    <input
      type="text"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      readOnly={readOnly}
      placeholder={placeholder}
      className={inputClass}
    />
  );
}

function NumberField({ value, onChange, readOnly }: {
  value: number; onChange: (v: number) => void; readOnly?: boolean;
}) {
  return (
    <input
      type="number"
      value={Number.isFinite(value) ? value : ""}
      onChange={(e) => onChange(e.target.value === "" ? 0 : Number(e.target.value))}
      readOnly={readOnly}
      className={inputClass}
    />
  );
}

function BooleanField({ value, onChange, readOnly }: {
  value: boolean; onChange: (v: boolean) => void; readOnly?: boolean;
}) {
  return (
    <input
      type="checkbox"
      checked={value}
      onChange={(e) => onChange(e.target.checked)}
      disabled={readOnly}
      className="h-4 w-4"
    />
  );
}

function SelectField({ options, value, onChange, readOnly }: {
  options: SelectOption[]; value: string; onChange: (v: string) => void; readOnly?: boolean;
}) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      disabled={readOnly}
      className={inputClass}
    >
      <option value="">—</option>
      {options.map((opt) => (
        <option key={opt.value} value={opt.value}>
          {opt.label ?? opt.value}
        </option>
      ))}
    </select>
  );
}

function MultiSelectField({ options, value, onChange, readOnly }: {
  options: SelectOption[]; value: string[]; onChange: (v: string[]) => void; readOnly?: boolean;
}) {
  return (
    <div className="flex flex-wrap gap-1">
      {options.map((opt) => {
        const selected = value.includes(opt.value);
        return (
          <button
            key={opt.value}
            type="button"
            disabled={readOnly}
            onClick={() => {
              const next = selected ? value.filter((v) => v !== opt.value) : [...value, opt.value];
              onChange(next);
            }}
            className={cn(
              "rounded border px-2 py-0.5 text-xs",
              selected ? "border-primary bg-primary/10 text-primary" : "border-border text-muted-foreground",
            )}
          >
            {opt.label ?? opt.value}
          </button>
        );
      })}
    </div>
  );
}

function DateField({ value, onChange, readOnly }: {
  value: string; onChange: (v: string) => void; readOnly?: boolean;
}) {
  return (
    <input
      type="date"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      readOnly={readOnly}
      className={inputClass}
    />
  );
}

function URLField({ value, onChange, readOnly }: {
  value: string; onChange: (v: string) => void; readOnly?: boolean;
}) {
  return (
    <input
      type="url"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      readOnly={readOnly}
      placeholder="https://…"
      className={inputClass}
    />
  );
}

// UserField renders a live UserPicker backed by /users/search. Column value
// is the numeric user id; 0 means unset.
function UserField({ value, onChange, readOnly }: {
  value: number; onChange: (v: number) => void; readOnly?: boolean;
}) {
  if (readOnly) {
    return <span className="text-sm">{value > 0 ? `user#${value}` : "—"}</span>;
  }
  return (
    <UserPicker
      value={value > 0 ? value : undefined}
      onPick={(u) => onChange(u ? u.id : 0)}
      placeholder="Pick a user…"
    />
  );
}

// BlockRefField looks up the target block's title / text live from the
// store so the chip re-renders when the target changes.
function BlockRefField({ value, onChange, readOnly }: {
  value: string; onChange: (v: string) => void; readOnly?: boolean;
}) {
  const target = useBlock(value || null);
  const display = target
    ? (((target.data as JSONMap).title as string | undefined) ?? target.text ?? value)
    : value || "pick block…";
  return (
    <input
      type="text"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      readOnly={readOnly}
      placeholder={display}
      className={inputClass}
    />
  );
}

function asString(v: unknown): string {
  return typeof v === "string" ? v : "";
}

function asNumber(v: unknown): number {
  return typeof v === "number" ? v : 0;
}

function asBoolean(v: unknown): boolean {
  return v === true;
}

function asStringArray(v: unknown): string[] {
  return Array.isArray(v) ? (v.filter((x) => typeof x === "string") as string[]) : [];
}

const inputClass =
  "w-full rounded border border-border bg-background px-2 py-1 text-sm outline-none focus:ring-1 focus:ring-ring";
