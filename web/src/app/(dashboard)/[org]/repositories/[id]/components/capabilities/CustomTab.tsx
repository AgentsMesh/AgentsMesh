import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

interface CustomTabProps {
  customName: string;
  setCustomName: (v: string) => void;
  customSlug: string;
  setCustomSlug: (v: string) => void;
  customTransport: string;
  setCustomTransport: (v: string) => void;
  customCommand: string;
  setCustomCommand: (v: string) => void;
  customArgs: string;
  setCustomArgs: (v: string) => void;
  customHttpUrl: string;
  setCustomHttpUrl: (v: string) => void;
  customEnvVars: Array<{ key: string; value: string }>;
  setCustomEnvVars: React.Dispatch<React.SetStateAction<Array<{ key: string; value: string }>>>;
  installing: boolean;
  t: (key: string) => string;
  onInstall: () => void;
}

export function CustomTab({
  customName, setCustomName, customSlug, setCustomSlug,
  customTransport, setCustomTransport, customCommand, setCustomCommand,
  customArgs, setCustomArgs, customHttpUrl, setCustomHttpUrl,
  customEnvVars, setCustomEnvVars, installing, t, onInstall,
}: CustomTabProps) {
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="text-sm font-medium mb-1 block">
            {t("extensions.serverName")} <span className="text-destructive">*</span>
          </label>
          <Input placeholder={t("extensions.serverNamePlaceholder")} value={customName}
            onChange={(e) => setCustomName(e.target.value)} />
        </div>
        <div>
          <label className="text-sm font-medium mb-1 block">
            {t("extensions.slug")} <span className="text-destructive">*</span>
          </label>
          <Input placeholder={t("extensions.slugPlaceholder")} value={customSlug}
            onChange={(e) => setCustomSlug(e.target.value)} />
        </div>
      </div>

      <TransportSelector value={customTransport} onChange={setCustomTransport} t={t} />

      {customTransport === "stdio" ? (
        <StdioFields command={customCommand} setCommand={setCustomCommand}
          args={customArgs} setArgs={setCustomArgs} t={t} />
      ) : (
        <div>
          <label className="text-sm font-medium mb-1 block">{t("extensions.httpUrl")}</label>
          <Input placeholder="http://localhost:3001/mcp" value={customHttpUrl}
            onChange={(e) => setCustomHttpUrl(e.target.value)} />
        </div>
      )}

      <EnvVarsEditor envVars={customEnvVars} setEnvVars={setCustomEnvVars} t={t} />

      <Button className="w-full" disabled={installing || !customName.trim() || !customSlug.trim()} onClick={onInstall}>
        {installing ? t("extensions.installing") : t("extensions.install")}
      </Button>
    </div>
  );
}

function TransportSelector({ value, onChange, t }: {
  value: string; onChange: (v: string) => void; t: (key: string) => string;
}) {
  return (
    <div>
      <label className="text-sm font-medium mb-1 block">{t("extensions.transportType")}</label>
      <div className="flex gap-2">
        {["stdio", "sse", "http"].map((tp) => (
          <Button key={tp} variant={value === tp ? "default" : "outline"} size="sm"
            onClick={() => onChange(tp)}>{tp}</Button>
        ))}
      </div>
    </div>
  );
}

function StdioFields({ command, setCommand, args, setArgs, t }: {
  command: string; setCommand: (v: string) => void;
  args: string; setArgs: (v: string) => void; t: (key: string) => string;
}) {
  return (
    <>
      <div>
        <label className="text-sm font-medium mb-1 block">{t("extensions.command")}</label>
        <Input placeholder="npx" value={command} onChange={(e) => setCommand(e.target.value)} />
      </div>
      <div>
        <label className="text-sm font-medium mb-1 block">{t("extensions.args")}</label>
        <Input placeholder="-y @modelcontextprotocol/server-filesystem /path" value={args}
          onChange={(e) => setArgs(e.target.value)} />
        <p className="text-xs text-muted-foreground mt-1">{t("extensions.argsHint")}</p>
      </div>
    </>
  );
}

function EnvVarsEditor({ envVars, setEnvVars, t }: {
  envVars: Array<{ key: string; value: string }>;
  setEnvVars: React.Dispatch<React.SetStateAction<Array<{ key: string; value: string }>>>;
  t: (key: string) => string;
}) {
  return (
    <div>
      <label className="text-sm font-medium mb-2 block">{t("extensions.envVars")}</label>
      {envVars.map((entry, idx) => (
        <div key={idx} className="flex gap-2 mb-2">
          <Input placeholder="KEY" value={entry.key}
            onChange={(e) => setEnvVars((prev) => prev.map((item, i) => (i === idx ? { ...item, key: e.target.value } : item)))} />
          <Input placeholder="value" value={entry.value}
            onChange={(e) => setEnvVars((prev) => prev.map((item, i) => (i === idx ? { ...item, value: e.target.value } : item)))} />
          <Button variant="ghost" size="sm" className="text-destructive"
            onClick={() => setEnvVars((prev) => prev.filter((_, i) => i !== idx))}>x</Button>
        </div>
      ))}
      <Button variant="outline" size="sm"
        onClick={() => setEnvVars((prev) => [...prev, { key: "", value: "" }])}>
        {t("extensions.addEnvVar")}
      </Button>
    </div>
  );
}
