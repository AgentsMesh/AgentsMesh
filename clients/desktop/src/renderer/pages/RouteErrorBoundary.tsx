import { useRouteError, Link, useNavigate, isRouteErrorResponse } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { AlertTriangle, Home, RefreshCw, ArrowLeft } from "lucide-react";
import { useTranslations } from "next-intl";

export function RouteErrorBoundary() {
  const error = useRouteError();
  const navigate = useNavigate();
  const t = useTranslations();

  const title = isRouteErrorResponse(error)
    ? `${error.status} ${error.statusText || t("errors.notFound")}`
    : t("errors.unexpected");
  const message = isRouteErrorResponse(error)
    ? error.data?.message || error.statusText
    : error instanceof Error
      ? error.message
      : String(error ?? "");

  return (
    <div className="flex h-screen w-screen items-center justify-center bg-background p-6">
      <div className="flex w-full max-w-md flex-col items-center gap-4 rounded-lg border border-border bg-card p-8 text-center shadow-sm">
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-destructive/10">
          <AlertTriangle className="h-6 w-6 text-destructive" />
        </div>
        <h1 className="text-lg font-semibold text-foreground">{title}</h1>
        {message && (
          <p className="max-w-sm break-words text-sm text-muted-foreground">{message}</p>
        )}
        <div className="mt-2 flex flex-wrap justify-center gap-2">
          <Button variant="outline" size="sm" onClick={() => navigate(-1)} className="gap-1.5">
            <ArrowLeft className="h-3.5 w-3.5" />
            {t("errors.goBack")}
          </Button>
          <Button variant="outline" size="sm" onClick={() => window.location.reload()} className="gap-1.5">
            <RefreshCw className="h-3.5 w-3.5" />
            {t("errors.reload")}
          </Button>
          <Button size="sm" asChild className="gap-1.5">
            <Link to="/">
              <Home className="h-3.5 w-3.5" />
              {t("errors.home")}
            </Link>
          </Button>
        </div>
      </div>
    </div>
  );
}
