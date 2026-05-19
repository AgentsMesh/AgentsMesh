import { app } from "electron";

// MUST be called synchronously before app.whenReady() — second-instance fires before
// whenReady on duplicate-launch path. Returns false on duplicate (also app.quit'd).
export function acquireSingleInstance(): boolean {
  if (!app.requestSingleInstanceLock()) {
    app.quit();
    return false;
  }
  return true;
}
