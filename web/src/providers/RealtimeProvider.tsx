/**
 * Re-export from modular realtime directory.
 * The monolithic RealtimeProvider has been split into domain-specific event
 * handlers under providers/realtime/ for better maintainability (SRP).
 */
export { RealtimeProvider, useRealtime } from "./realtime";
