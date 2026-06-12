export function StatusPill({ label, tone }: { label: string; tone: string }) {
  return (
    <span className={`shrink-0 rounded px-1.5 py-0.5 text-[10px] font-medium ${tone}`}>
      {label}
    </span>
  );
}
