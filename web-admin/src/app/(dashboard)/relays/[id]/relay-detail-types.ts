import type { ActiveSession, RelayInfo } from "@/lib/api/admin";

export interface RelayDetailHeaderProps {
  relay: RelayInfo;
  healthyRelays: RelayInfo[];
  isUnregistering: boolean;
  onUnregister: (migrate: boolean) => void;
  onBack: () => void;
}

export interface RelayInfoCardsProps {
  relay: RelayInfo;
  sessionCount: number;
}

export interface RelaySessionsTableProps {
  sessions: ActiveSession[];
  healthyRelays: RelayInfo[];
  migratingPod: string | null;
  targetRelay: string;
  isMigratingSession: boolean;
  onSetTargetRelay: (v: string) => void;
  onMigrate: (session: ActiveSession) => void;
  onCancelMigrate: () => void;
}
