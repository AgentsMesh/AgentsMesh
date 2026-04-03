import { ImageResponse } from "next/og";

export const runtime = "edge";
export const alt = "AgentsMesh - The Agent Workforce Platform | Ship like a team of fifty";
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";

export default async function Image() {
  return new ImageResponse(
    (
      <div
        style={{
          background: "linear-gradient(135deg, #0a0a0a 0%, #1a1a2e 50%, #0a0a0a 100%)",
          width: "100%",
          height: "100%",
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          justifyContent: "center",
          fontFamily: "system-ui, sans-serif",
          position: "relative",
          overflow: "hidden",
        }}
      >
        {/* Grid pattern */}
        <div
          style={{
            position: "absolute",
            inset: 0,
            opacity: 0.08,
            backgroundImage:
              "linear-gradient(rgba(34,211,238,0.5) 1px, transparent 1px), linear-gradient(90deg, rgba(34,211,238,0.5) 1px, transparent 1px)",
            backgroundSize: "60px 60px",
          }}
        />
        {/* Glow */}
        <div
          style={{
            position: "absolute",
            top: "50%",
            left: "50%",
            transform: "translate(-50%, -50%)",
            width: 600,
            height: 600,
            borderRadius: "50%",
            background: "radial-gradient(circle, rgba(34,211,238,0.15) 0%, transparent 70%)",
          }}
        />
        {/* Logo text */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 16,
            marginBottom: 32,
          }}
        >
          <div
            style={{
              width: 64,
              height: 64,
              borderRadius: 16,
              background: "linear-gradient(135deg, #22d3ee, #0ea5e9)",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              fontSize: 32,
              fontWeight: 800,
              color: "#0a0a0a",
            }}
          >
            AM
          </div>
          <span style={{ fontSize: 48, fontWeight: 800, color: "#ededed" }}>
            AgentsMesh
          </span>
        </div>
        {/* Tagline */}
        <div
          style={{
            fontSize: 28,
            color: "#a1a1aa",
            marginBottom: 16,
            textAlign: "center",
          }}
        >
          The Agent Workforce Platform
        </div>
        {/* Slogan */}
        <div
          style={{
            fontSize: 36,
            fontWeight: 700,
            textAlign: "center",
            maxWidth: 900,
            lineHeight: 1.3,
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
          }}
        >
          <span style={{ color: "#ededed" }}>Ship like a team of fifty.</span>
          <span
            style={{
              background: "linear-gradient(90deg, #22d3ee, #0ea5e9)",
              backgroundClip: "text",
              color: "transparent",
            }}
          >
            With a team of five.
          </span>
        </div>
      </div>
    ),
    { ...size },
  );
}
