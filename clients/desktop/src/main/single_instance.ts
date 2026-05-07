import { app } from "electron";

/**
 * MUST be called synchronously before app.whenReady() — second-instance
 * fires before whenReady on the duplicate-launch path. Returns false on
 * the duplicate (which is also app.quit'd here) so callers can skip
 * setup that would race the dying process.
 */
export function acquireSingleInstance(): boolean {
  if (!app.requestSingleInstanceLock()) {
    app.quit();
    return false;
  }
  return true;
}
