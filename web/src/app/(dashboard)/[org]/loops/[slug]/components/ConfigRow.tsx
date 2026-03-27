interface ConfigRowProps {
  icon: React.ReactNode;
  label: string;
  value: React.ReactNode;
}

export function ConfigRow({ icon, label, value }: ConfigRowProps) {
  return (
    <div className="flex items-center justify-between text-sm">
      <span className="flex items-center gap-2 text-muted-foreground">
        {icon}
        {label}
      </span>
      <span className="font-medium capitalize min-w-0">{value}</span>
    </div>
  );
}
