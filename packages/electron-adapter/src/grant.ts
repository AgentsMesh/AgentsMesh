// grant has no legacy JSON surface and no cached state — all three verbs
// (createGrant / deleteGrant / listGrants) go through the Connect fallback in
// provider.ts. An empty proxy target is all that's needed; the renderer was
// previously getting a no-op stub because the desktop provider never wired
// grantService at all.
export class ElectronGrantService {}
