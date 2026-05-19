// Auth route group layout. Pages here are wasm-zero — login, register,
// OAuth callbacks, verify-email, forgot/reset password, invite acceptance,
// onboarding, and runner authorization all go through @/lib/light-auth and
// read session state via useLightSession/useRedirectIfAuthenticated/
// useRequireLightAuth. The 40MB wasm bundle is deferred until the user
// crosses into (dashboard), so the time-to-login form stays sub-second
// even on slow connections.
// MUST NOT wrap this group in AuthBootstrap/WasmProvider — that would
// undo the entire optimization. The corresponding negative check lives in
// clients/web/scripts/check-no-wasm-in-marketing.sh.
export default function AuthLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <>{children}</>;
}
