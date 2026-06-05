import React from "react";

interface State {
  error: Error | null;
}

// RouterProvider's per-route errorElement cannot catch throws from the
// providers mounted ABOVE it (AppProviders / QueryClient) nor from an
// errorElement that itself throws — those unwind to the root and blank the
// window. This boundary is the last resort. It deliberately uses inline styles
// and hard-coded copy: ThemeProvider / Tailwind / next-intl may be the very
// thing that crashed, so the fallback must not depend on any of them.
export class RootErrorBoundary extends React.Component<
  { children: React.ReactNode },
  State
> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo): void {
    console.error("[root-error-boundary]", error, info.componentStack);
  }

  render(): React.ReactNode {
    if (!this.state.error) return this.props.children;
    return (
      <div style={WRAP}>
        <div style={CARD}>
          <h1 style={TITLE}>Something went wrong</h1>
          <p style={MESSAGE}>{this.state.error.message}</p>
          <button style={BUTTON} onClick={() => window.location.reload()}>
            Reload
          </button>
        </div>
      </div>
    );
  }
}

const WRAP: React.CSSProperties = {
  display: "flex", height: "100vh", width: "100vw",
  alignItems: "center", justifyContent: "center",
  background: "#0a0a0a", color: "#e5e5e5",
  fontFamily: "system-ui, -apple-system, sans-serif", padding: 24,
};
const CARD: React.CSSProperties = { maxWidth: 420, textAlign: "center" };
const TITLE: React.CSSProperties = { fontSize: 18, fontWeight: 600, marginBottom: 8 };
const MESSAGE: React.CSSProperties = {
  fontSize: 13, color: "#a3a3a3", marginBottom: 20, wordBreak: "break-word",
};
const BUTTON: React.CSSProperties = {
  padding: "8px 16px", fontSize: 13, borderRadius: 6,
  border: "1px solid #404040", background: "#171717",
  color: "#e5e5e5", cursor: "pointer",
};
