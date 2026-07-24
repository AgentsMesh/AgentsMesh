import { RelayPodSession } from "./relayPodSession";

export class RelayPodRegistry {
  private readonly sessions = new Map<string, RelayPodSession>();

  get(podKey: string): RelayPodSession | undefined {
    return this.sessions.get(podKey);
  }

  getOrCreate(podKey: string): RelayPodSession {
    let session = this.sessions.get(podKey);
    if (!session) {
      session = new RelayPodSession(podKey);
      this.sessions.set(podKey, session);
    }
    return session;
  }

  handleDisconnected(podKey: string): boolean {
    const session = this.sessions.get(podKey);
    const keepSession = session?.handlePodDisconnected() === true;
    if (session && !keepSession) this.sessions.delete(podKey);
    return keepSession;
  }

  clear(podKey: string): void {
    this.sessions.get(podKey)?.clear();
    this.sessions.delete(podKey);
  }

  clearAll(): void {
    for (const session of this.sessions.values()) session.clear();
    this.sessions.clear();
  }
}
