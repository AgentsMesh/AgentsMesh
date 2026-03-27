import { Button } from "@/components/ui/button";

interface SetupStepsProps {
  token: string | null;
  tokenCopied: boolean;
  loading: boolean;
  serverUrl: string;
  registerMethod: "token" | "login";
  setRegisterMethod: (method: "token" | "login") => void;
  connectionStatus: "waiting" | "connected" | "timeout";
  waitTime: number;
  t: (key: string) => string;
  onCopyToken: () => void;
  onRetry: () => void;
}

export function SetupSteps({
  token, tokenCopied, loading, serverUrl, registerMethod, setRegisterMethod,
  connectionStatus, waitTime, t, onCopyToken, onRetry,
}: SetupStepsProps) {
  return (
    <div className="space-y-6">
      <TokenStep token={token} tokenCopied={tokenCopied} loading={loading} t={t} onCopyToken={onCopyToken} />
      <InstallStep serverUrl={serverUrl} token={token} registerMethod={registerMethod}
        setRegisterMethod={setRegisterMethod} t={t} />
      <WaitingStep connectionStatus={connectionStatus} waitTime={waitTime} t={t} onRetry={onRetry} />
    </div>
  );
}

function TokenStep({ token, tokenCopied, loading, t, onCopyToken }: {
  token: string | null; tokenCopied: boolean; loading: boolean;
  t: (key: string) => string; onCopyToken: () => void;
}) {
  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <StepNumber num={1} />
        <h3 className="font-medium text-foreground">{t("auth.onboarding.localRunner.step1Title")}</h3>
      </div>
      <div className="ml-8">
        {loading ? (
          <div className="p-3 bg-muted rounded-md text-sm text-muted-foreground">
            {t("auth.onboarding.localRunner.generatingToken")}
          </div>
        ) : token ? (
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <code className="flex-1 p-3 bg-muted rounded-md text-sm font-mono text-foreground overflow-x-auto">{token}</code>
              <Button variant="outline" size="sm" onClick={onCopyToken} className="flex-shrink-0">
                {tokenCopied ? (
                  <svg className="w-4 h-4 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                ) : (
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                  </svg>
                )}
              </Button>
            </div>
            <p className="text-xs text-amber-600 dark:text-amber-400">
              {t("auth.onboarding.localRunner.tokenWarning")}
            </p>
          </div>
        ) : (
          <div className="p-3 bg-destructive/10 rounded-md text-sm text-destructive">
            {t("auth.onboarding.localRunner.tokenGenerationFailedShort")}
          </div>
        )}
      </div>
    </div>
  );
}

function InstallStep({ serverUrl, token, registerMethod, setRegisterMethod, t }: {
  serverUrl: string; token: string | null; registerMethod: "token" | "login";
  setRegisterMethod: (method: "token" | "login") => void; t: (key: string) => string;
}) {
  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <StepNumber num={2} />
        <h3 className="font-medium text-foreground">{t("auth.onboarding.localRunner.step2Title")}</h3>
      </div>
      <div className="ml-8 space-y-3">
        <div className="p-4 bg-muted rounded-md">
          <p className="text-xs text-muted-foreground mb-2"># macOS / Linux</p>
          <code className="text-sm font-mono text-foreground block">curl -fsSL {serverUrl}/install.sh | sh</code>
        </div>
        <div className="p-4 bg-muted rounded-md">
          <p className="text-xs text-muted-foreground mb-2"># Windows (PowerShell)</p>
          <code className="text-sm font-mono text-foreground block">irm {serverUrl}/install.ps1 | iex</code>
        </div>
        <RegisterMethodTabs method={registerMethod} setMethod={setRegisterMethod}
          serverUrl={serverUrl} token={token} t={t} />
      </div>
    </div>
  );
}

function RegisterMethodTabs({ method, setMethod, serverUrl, token, t }: {
  method: "token" | "login"; setMethod: (m: "token" | "login") => void;
  serverUrl: string; token: string | null; t: (key: string) => string;
}) {
  return (
    <div className="border border-border rounded-md overflow-hidden">
      <div className="flex border-b border-border">
        {(["token", "login"] as const).map((m) => (
          <button key={m} type="button"
            className={`flex-1 px-4 py-2 text-sm font-medium transition-colors ${
              method === m ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground hover:text-foreground"
            }`}
            onClick={() => setMethod(m)}>
            {t(`auth.onboarding.localRunner.method${m === "token" ? "Token" : "Login"}`)}
          </button>
        ))}
      </div>
      <div className="p-4 bg-muted">
        {method === "token" ? (
          <>
            <p className="text-xs text-muted-foreground mb-2"># {t("auth.onboarding.localRunner.startRunnerComment")}</p>
            <code className="text-sm font-mono text-foreground block whitespace-pre-wrap">
{`agentsmesh-runner register --server ${serverUrl} --token ${token || "<your-token>"}
agentsmesh-runner run`}
            </code>
          </>
        ) : (
          <>
            <p className="text-xs text-muted-foreground mb-2"># {t("auth.onboarding.localRunner.loginRunnerComment")}</p>
            <code className="text-sm font-mono text-foreground block whitespace-pre-wrap">
{`agentsmesh-runner login
agentsmesh-runner run`}
            </code>
          </>
        )}
      </div>
    </div>
  );
}

function WaitingStep({ connectionStatus, waitTime, t, onRetry }: {
  connectionStatus: "waiting" | "connected" | "timeout"; waitTime: number;
  t: (key: string) => string; onRetry: () => void;
}) {
  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, "0")}`;
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <StepNumber num={3} />
        <h3 className="font-medium text-foreground">{t("auth.onboarding.localRunner.step3Title")}</h3>
      </div>
      <div className="ml-8">
        {connectionStatus === "waiting" && (
          <div className="p-4 border border-border rounded-md">
            <div className="flex items-center gap-3">
              <div className="w-5 h-5 border-2 border-primary border-t-transparent rounded-full animate-spin" />
              <div>
                <p className="text-sm text-foreground">{t("auth.onboarding.localRunner.waitingForConnection")}</p>
                <p className="text-xs text-muted-foreground">{t("auth.onboarding.localRunner.elapsed")}: {formatTime(waitTime)}</p>
              </div>
            </div>
          </div>
        )}
        {connectionStatus === "timeout" && (
          <div className="p-4 border border-amber-500/50 bg-amber-50 dark:bg-amber-950/30 rounded-md">
            <p className="text-sm text-amber-800 dark:text-amber-200">{t("auth.onboarding.localRunner.connectionTimeout")}</p>
            <Button variant="outline" size="sm" className="mt-2" onClick={onRetry}>
              {t("auth.onboarding.localRunner.retry")}
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}

function StepNumber({ num }: { num: number }) {
  return (
    <div className="w-6 h-6 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-sm font-medium">
      {num}
    </div>
  );
}
