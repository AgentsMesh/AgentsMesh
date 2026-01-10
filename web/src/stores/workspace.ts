import { create } from "zustand";
import { persist } from "zustand/middleware";

/**
 * Terminal pane configuration
 */
export interface TerminalPane {
  id: string;
  podKey: string;
  title: string;
  isActive: boolean;
  // Grid position for desktop
  gridPosition?: {
    x: number;
    y: number;
    w: number;
    h: number;
  };
}

/**
 * Grid layout configuration
 */
export type GridLayoutType = "1x1" | "1x2" | "2x1" | "2x2" | "custom";

export interface GridLayout {
  type: GridLayoutType;
  rows: number;
  cols: number;
}

/**
 * Workspace state management
 */
interface WorkspaceState {
  // Terminal panes
  panes: TerminalPane[];
  activePane: string | null;

  // Grid layout
  gridLayout: GridLayout;

  // Mobile state
  mobileActiveIndex: number;

  // Terminal settings
  terminalFontSize: number;

  // Actions
  addPane: (podKey: string, title?: string) => string;
  removePane: (paneId: string) => void;
  setActivePane: (paneId: string | null) => void;
  updatePanePosition: (paneId: string, position: TerminalPane["gridPosition"]) => void;
  setGridLayout: (layout: GridLayout) => void;
  setMobileActiveIndex: (index: number) => void;
  setTerminalFontSize: (size: number) => void;
  clearAllPanes: () => void;
  getPaneByPodKey: (podKey: string) => TerminalPane | undefined;

  // Hydration
  _hasHydrated: boolean;
  setHasHydrated: (state: boolean) => void;
}

// Generate unique pane ID
const generatePaneId = () => `pane-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

export const useWorkspaceStore = create<WorkspaceState>()(
  persist(
    (set, get) => ({
      // Initial state
      panes: [],
      activePane: null,
      gridLayout: { type: "1x1", rows: 1, cols: 1 },
      mobileActiveIndex: 0,
      terminalFontSize: 14,
      _hasHydrated: false,

      // Actions
      addPane: (podKey, title) => {
        const existingPane = get().panes.find((p) => p.podKey === podKey);
        if (existingPane) {
          set({ activePane: existingPane.id });
          return existingPane.id;
        }

        const id = generatePaneId();
        const panes = get().panes;
        const newPane: TerminalPane = {
          id,
          podKey,
          title: title || `Pod ${podKey.substring(0, 8)}`,
          isActive: true,
          gridPosition: {
            x: panes.length % 2,
            y: Math.floor(panes.length / 2),
            w: 1,
            h: 1,
          },
        };

        set((state) => ({
          panes: [...state.panes.map((p) => ({ ...p, isActive: false })), newPane],
          activePane: id,
        }));

        return id;
      },

      removePane: (paneId) => {
        set((state) => {
          const newPanes = state.panes.filter((p) => p.id !== paneId);
          const wasActive = state.activePane === paneId;
          return {
            panes: newPanes,
            activePane: wasActive ? (newPanes[0]?.id || null) : state.activePane,
            mobileActiveIndex: Math.min(state.mobileActiveIndex, Math.max(0, newPanes.length - 1)),
          };
        });
      },

      setActivePane: (paneId) => {
        set((state) => ({
          panes: state.panes.map((p) => ({
            ...p,
            isActive: p.id === paneId,
          })),
          activePane: paneId,
        }));
      },

      updatePanePosition: (paneId, position) => {
        set((state) => ({
          panes: state.panes.map((p) =>
            p.id === paneId ? { ...p, gridPosition: position } : p
          ),
        }));
      },

      setGridLayout: (layout) => {
        set({ gridLayout: layout });
      },

      setMobileActiveIndex: (index) => {
        const panes = get().panes;
        if (index >= 0 && index < panes.length) {
          set({
            mobileActiveIndex: index,
            activePane: panes[index]?.id || null,
          });
        }
      },

      setTerminalFontSize: (size) => {
        set({ terminalFontSize: Math.min(Math.max(size, 10), 24) });
      },

      clearAllPanes: () => {
        set({ panes: [], activePane: null, mobileActiveIndex: 0 });
      },

      getPaneByPodKey: (podKey) => {
        return get().panes.find((p) => p.podKey === podKey);
      },

      setHasHydrated: (state) => {
        set({ _hasHydrated: state });
      },
    }),
    {
      name: "agentmesh-workspace",
      partialize: (state) => ({
        panes: state.panes,
        activePane: state.activePane,
        gridLayout: state.gridLayout,
        terminalFontSize: state.terminalFontSize,
      }),
      onRehydrateStorage: () => (state) => {
        state?.setHasHydrated(true);
      },
    }
  )
);

/**
 * Terminal connection pool for managing WebSocket connections
 */
interface TerminalConnection {
  ws: WebSocket;
  podKey: string;
  buffer: Uint8Array[];
  status: "connecting" | "connected" | "disconnected" | "error";
  lastActivity: number;
  listeners: Set<(data: Uint8Array | string) => void>;
  reconnectAttempts: number;
  reconnectTimer: ReturnType<typeof setTimeout> | null;
}

class TerminalConnectionPool {
  private connections: Map<string, TerminalConnection> = new Map();
  private maxBufferSize = 100; // Keep last 100 messages for reconnection
  private maxReconnectAttempts = 5;
  private baseReconnectDelay = 1000;

  getConnection(podKey: string): TerminalConnection | undefined {
    return this.connections.get(podKey);
  }

  connect(
    podKey: string,
    onMessage: (data: Uint8Array | string) => void
  ): { send: (data: string) => void; disconnect: () => void } {
    let conn = this.connections.get(podKey);

    if (conn) {
      // Add listener to existing connection
      conn.listeners.add(onMessage);

      // Replay buffer
      for (const data of conn.buffer) {
        onMessage(data);
      }

      return {
        send: (data: string) => this.send(podKey, data),
        disconnect: () => this.removeListener(podKey, onMessage),
      };
    }

    // Create new connection
    // Derive WebSocket URL from API URL or use explicit WS URL
    let wsUrl = process.env.NEXT_PUBLIC_WS_URL;
    if (!wsUrl) {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL;
      if (apiUrl) {
        // Convert http(s):// to ws(s)://
        wsUrl = apiUrl.replace(/^http/, "ws");
      } else if (typeof window !== "undefined") {
        // Fallback: derive from current page location
        const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
        const host = window.location.hostname;
        const port = "8080"; // Default API port
        wsUrl = `${protocol}//${host}:${port}`;
      } else {
        wsUrl = "ws://localhost:8080";
      }
    }
    let token = null;
    let orgSlug = null;

    try {
      const authData = localStorage.getItem("agentmesh-auth");
      if (authData) {
        const parsed = JSON.parse(authData);
        token = parsed.state?.token;
        orgSlug = parsed.state?.currentOrg?.slug;
      }
    } catch (e) {
      console.error("Failed to parse auth data:", e);
    }

    const ws = new WebSocket(
      `${wsUrl}/api/v1/orgs/${orgSlug}/ws/terminal/${podKey}?token=${token}`
    );
    ws.binaryType = "arraybuffer";

    conn = {
      ws,
      podKey,
      buffer: [],
      status: "connecting",
      lastActivity: Date.now(),
      listeners: new Set([onMessage]),
      reconnectAttempts: 0,
      reconnectTimer: null,
    };

    this.connections.set(podKey, conn);

    ws.onopen = () => {
      const c = this.connections.get(podKey);
      if (c) {
        c.status = "connected";
        c.lastActivity = Date.now();
        c.reconnectAttempts = 0; // Reset on successful connection
      }
    };

    ws.onmessage = (event) => {
      const c = this.connections.get(podKey);
      if (c) {
        c.lastActivity = Date.now();

        let data: Uint8Array | string;
        if (event.data instanceof ArrayBuffer) {
          data = new Uint8Array(event.data);
        } else {
          data = event.data;
        }

        // Buffer the message
        if (typeof data !== "string") {
          c.buffer.push(data);
          if (c.buffer.length > this.maxBufferSize) {
            c.buffer.shift();
          }
        }

        // Notify all listeners
        for (const listener of c.listeners) {
          listener(data);
        }
      }
    };

    ws.onerror = (error) => {
      console.error(`Terminal WebSocket error for ${podKey}:`, error);
      const c = this.connections.get(podKey);
      if (c) {
        c.status = "error";
        // Error often precedes close, but schedule reconnect just in case
        // onclose will also schedule reconnect, but scheduleReconnect handles duplicates
        if (c.listeners.size > 0 && !c.reconnectTimer) {
          this.scheduleReconnect(podKey);
        }
      }
    };

    ws.onclose = () => {
      const c = this.connections.get(podKey);
      if (c) {
        c.status = "disconnected";
        // Auto-reconnect if there are still listeners
        if (c.listeners.size > 0) {
          this.scheduleReconnect(podKey);
        }
      }
    };

    return {
      send: (data: string) => this.send(podKey, data),
      disconnect: () => this.removeListener(podKey, onMessage),
    };
  }

  send(podKey: string, data: string) {
    const conn = this.connections.get(podKey);
    if (conn && conn.ws.readyState === WebSocket.OPEN) {
      conn.ws.send(JSON.stringify({ type: "input", data }));
      conn.lastActivity = Date.now();
    }
  }

  sendResize(podKey: string, rows: number, cols: number) {
    const conn = this.connections.get(podKey);
    if (conn && conn.ws.readyState === WebSocket.OPEN) {
      conn.ws.send(JSON.stringify({ type: "resize", rows, cols }));
    }
  }

  removeListener(podKey: string, listener: (data: Uint8Array | string) => void) {
    const conn = this.connections.get(podKey);
    if (conn) {
      conn.listeners.delete(listener);

      // Delay disconnect to handle React Strict Mode double-invoke
      // If a new listener is added within the delay, don't disconnect
      if (conn.listeners.size === 0) {
        setTimeout(() => {
          const currentConn = this.connections.get(podKey);
          if (currentConn && currentConn.listeners.size === 0) {
            this.disconnect(podKey);
          }
        }, 100);
      }
    }
  }

  disconnect(podKey: string) {
    const conn = this.connections.get(podKey);
    if (conn) {
      // Clear any pending reconnect timer
      if (conn.reconnectTimer) {
        clearTimeout(conn.reconnectTimer);
        conn.reconnectTimer = null;
      }
      // Only close if WebSocket is actually open or connecting
      if (conn.ws.readyState === WebSocket.OPEN || conn.ws.readyState === WebSocket.CONNECTING) {
        conn.ws.close();
      }
      this.connections.delete(podKey);
    }
  }

  private scheduleReconnect(podKey: string) {
    const conn = this.connections.get(podKey);
    if (!conn) return;

    if (conn.reconnectAttempts >= this.maxReconnectAttempts) {
      console.log(`Max reconnect attempts reached for ${podKey}`);
      return;
    }

    // Exponential backoff with cap at 30 seconds
    const delay = Math.min(
      this.baseReconnectDelay * Math.pow(2, conn.reconnectAttempts),
      30000
    );

    console.log(`Scheduling reconnect for ${podKey} in ${delay}ms (attempt ${conn.reconnectAttempts + 1}/${this.maxReconnectAttempts})`);

    conn.reconnectTimer = setTimeout(() => {
      conn.reconnectAttempts++;
      this.reconnect(podKey);
    }, delay);
  }

  private reconnect(podKey: string) {
    const oldConn = this.connections.get(podKey);
    if (!oldConn || oldConn.listeners.size === 0) return;

    console.log(`Reconnecting terminal for ${podKey}...`);

    // Save existing listeners and buffer
    const listeners = new Set(oldConn.listeners);
    const buffer = [...oldConn.buffer];
    const reconnectAttempts = oldConn.reconnectAttempts;

    // Clean up old connection
    if (oldConn.ws.readyState === WebSocket.OPEN || oldConn.ws.readyState === WebSocket.CONNECTING) {
      oldConn.ws.close();
    }
    this.connections.delete(podKey);

    // Reconnect with first listener to trigger connection
    const firstListener = listeners.values().next().value;
    if (firstListener) {
      this.connect(podKey, firstListener);

      // Restore state to new connection
      const newConn = this.connections.get(podKey);
      if (newConn) {
        // Add remaining listeners
        listeners.forEach((l) => {
          if (l !== firstListener) newConn.listeners.add(l);
        });
        // Restore reconnect attempts counter
        newConn.reconnectAttempts = reconnectAttempts;
        // Restore buffer
        newConn.buffer = buffer;
      }
    }
  }

  disconnectAll() {
    for (const [podKey] of this.connections) {
      this.disconnect(podKey);
    }
  }

  getStatus(podKey: string): TerminalConnection["status"] | "none" {
    return this.connections.get(podKey)?.status || "none";
  }

  isConnected(podKey: string): boolean {
    const conn = this.connections.get(podKey);
    return conn?.status === "connected" && conn.ws.readyState === WebSocket.OPEN;
  }
}

// Singleton instance
export const terminalPool = new TerminalConnectionPool();
