// Facade re-export of the loop Connect-RPC adapter. Business code imports
// from here (or from the `@/lib/api` barrel) so the wire-shape layer stays
// internal to the facade boundary. Tests mock this path.

export {
  listLoops,
  listLoopsRaw,
  getLoop,
  getLoopRaw,
  createLoop,
  updateLoop,
  deleteLoop,
  enableLoop,
  disableLoop,
  triggerLoop,
  listLoopRuns,
  listRunsRaw,
  cancelLoopRun,
  type TriggerLoopResult,
} from "../connect/loopConnect";
