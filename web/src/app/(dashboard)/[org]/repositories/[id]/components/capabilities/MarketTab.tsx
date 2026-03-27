import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { ExternalLink, Search } from "lucide-react";
import type { McpMarketItem } from "@/lib/api";

interface MarketTabProps {
  marketServers: McpMarketItem[];
  marketQuery: string;
  setMarketQuery: (q: string) => void;
  loadingMarket: boolean;
  selectedTemplate: McpMarketItem | null;
  envVars: Record<string, string>;
  setEnvVars: React.Dispatch<React.SetStateAction<Record<string, string>>>;
  hasUnfilledRequiredEnvVars: boolean;
  installing: boolean;
  t: (key: string) => string;
  onSearch: () => void;
  onSelectTemplate: (item: McpMarketItem) => void;
  onClearTemplate: () => void;
  onInstall: () => void;
}

export function MarketTab({
  marketServers, marketQuery, setMarketQuery, loadingMarket,
  selectedTemplate, envVars, setEnvVars, hasUnfilledRequiredEnvVars,
  installing, t, onSearch, onSelectTemplate, onClearTemplate, onInstall,
}: MarketTabProps) {
  if (selectedTemplate) {
    return (
      <TemplateConfigForm
        template={selectedTemplate} envVars={envVars} setEnvVars={setEnvVars}
        hasUnfilledRequiredEnvVars={hasUnfilledRequiredEnvVars}
        installing={installing} t={t} onClear={onClearTemplate} onInstall={onInstall} />
    );
  }

  return (
    <>
      <div className="flex gap-2 mb-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input className="pl-9" placeholder={t("extensions.searchMcpServers")}
            value={marketQuery} onChange={(e) => setMarketQuery(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && onSearch()} />
        </div>
        <Button variant="outline" onClick={onSearch}>{t("extensions.search")}</Button>
      </div>
      {loadingMarket ? (
        <div className="py-8 text-center">
          <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary mx-auto"></div>
        </div>
      ) : marketServers.length === 0 ? (
        <p className="text-sm text-muted-foreground text-center py-8">{t("extensions.noMarketMcpServers")}</p>
      ) : (
        <div className="space-y-2 max-h-80 overflow-y-auto">
          {marketServers.map((item) => (
            <MarketItemCard key={item.id} item={item} t={t} onSelect={() => onSelectTemplate(item)} />
          ))}
        </div>
      )}
    </>
  );
}

function MarketItemCard({ item, t, onSelect }: { item: McpMarketItem; t: (key: string) => string; onSelect: () => void }) {
  return (
    <div className="border border-border rounded-lg p-3 flex items-center justify-between gap-3 cursor-pointer hover:bg-muted/50"
      onClick={onSelect}>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="font-medium">{item.name}</span>
          <Badge variant="secondary" className="text-xs">{item.transport_type}</Badge>
          {item.category && <Badge variant="outline" className="text-xs">{item.category}</Badge>}
          {item.repository_url && (
            <a href={item.repository_url} target="_blank" rel="noopener noreferrer"
              className="text-muted-foreground hover:text-foreground shrink-0"
              title={t("extensions.viewSource")} onClick={(e) => e.stopPropagation()}>
              <ExternalLink className="w-3.5 h-3.5" />
            </a>
          )}
        </div>
        {item.description && (
          <p className="text-xs text-muted-foreground mt-1 line-clamp-2">{item.description}</p>
        )}
      </div>
      <Button size="sm" variant="outline">{t("extensions.select")}</Button>
    </div>
  );
}

function TemplateConfigForm({ template, envVars, setEnvVars, hasUnfilledRequiredEnvVars, installing, t, onClear, onInstall }: {
  template: McpMarketItem; envVars: Record<string, string>;
  setEnvVars: React.Dispatch<React.SetStateAction<Record<string, string>>>;
  hasUnfilledRequiredEnvVars: boolean; installing: boolean;
  t: (key: string) => string; onClear: () => void; onInstall: () => void;
}) {
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2 mb-2">
        <span className="font-medium">{template.name}</span>
        <Badge variant="secondary" className="text-xs">{template.transport_type}</Badge>
        {template.repository_url && (
          <a href={template.repository_url} target="_blank" rel="noopener noreferrer"
            className="text-muted-foreground hover:text-foreground" title={t("extensions.viewSource")}>
            <ExternalLink className="w-3.5 h-3.5" />
          </a>
        )}
        <Button variant="ghost" size="sm" onClick={onClear}>{t("extensions.changeTemplate")}</Button>
      </div>
      {template.description && <p className="text-sm text-muted-foreground">{template.description}</p>}
      {(template.env_var_schema?.length ?? 0) > 0 && (
        <div className="space-y-3">
          <h4 className="text-sm font-medium">{t("extensions.envVars")}</h4>
          {template.env_var_schema!.map((entry) => (
            <div key={entry.name}>
              <label className="text-sm font-medium mb-1 block">
                {entry.label || entry.name}
                {entry.required && <span className="text-destructive ml-1">*</span>}
              </label>
              <Input type={entry.sensitive ? "password" : "text"}
                placeholder={entry.placeholder || entry.name} value={envVars[entry.name] || ""}
                onChange={(e) => setEnvVars((prev) => ({ ...prev, [entry.name]: e.target.value }))} />
            </div>
          ))}
        </div>
      )}
      <Button className="w-full" disabled={installing || hasUnfilledRequiredEnvVars} onClick={onInstall}>
        {installing ? t("extensions.installing") : t("extensions.install")}
      </Button>
    </div>
  );
}
