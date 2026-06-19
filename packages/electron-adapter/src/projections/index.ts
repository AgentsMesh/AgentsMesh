// Aggregated proto→viewModel projections, exposed as the
// `@agentsmesh/electron-adapter/projections` subpath so web can re-use them
// without pulling the Electron service classes (and their electron-only
// import graph, e.g. pod.ts → proto/pod_state) through the package's
// top-level entry.
export { loopToCache, loopRunToCache } from "./loop";
export {
  ticketToCache,
  boardColumnToCache,
  labelToCache,
  cacheTicketToProto,
} from "./ticket";
export { podToCache } from "./pod";
export { runnerToCache } from "./runner";
export { repositoryToCache, branchToCache } from "./repository";
