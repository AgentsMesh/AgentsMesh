export interface PodDriverDisconnected {
  podKey: string;
  generation: number;
}

export interface PodStatusEvent {
  generation: number;
  revision: number;
}

export interface PodAcpEvent {
  generation: number;
}

export function parsePodDriverDisconnected(raw: string): PodDriverDisconnected | null {
  try {
    const parsed = JSON.parse(raw) as Partial<PodDriverDisconnected>;
    if (
      typeof parsed.podKey !== "string"
      || !Number.isSafeInteger(parsed.generation)
      || parsed.generation! <= 0
    ) return null;
    return { podKey: parsed.podKey, generation: parsed.generation } as PodDriverDisconnected;
  } catch {
    return null;
  }
}

export function parsePodStatusEvent(raw: string): PodStatusEvent | null {
  try {
    const parsed = JSON.parse(raw) as Partial<PodStatusEvent>;
    if (
      !Number.isSafeInteger(parsed.generation)
      || parsed.generation! <= 0
      || !Number.isSafeInteger(parsed.revision)
      || parsed.revision! < 0
    ) return null;
    return { generation: parsed.generation, revision: parsed.revision } as PodStatusEvent;
  } catch {
    return null;
  }
}

export function parsePodAcpEvent(raw: string): PodAcpEvent | null {
  try {
    const parsed = JSON.parse(raw) as Partial<PodAcpEvent>;
    if (!Number.isSafeInteger(parsed.generation) || parsed.generation! <= 0) return null;
    return { generation: parsed.generation } as PodAcpEvent;
  } catch {
    return null;
  }
}
