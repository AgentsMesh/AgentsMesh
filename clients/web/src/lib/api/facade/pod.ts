// Pod view-model types moved to the zero-dep @agentsmesh/service-interface
// contract layer so the web fromProtoPod projection and the desktop
// electron-adapter projection share one definition. Re-exported here to
// preserve existing `@/lib/api/facade/pod` import paths.
export type { PodData } from "@agentsmesh/service-interface";
