import { publicRequest } from "./base";
import type { PublicPricingResponse, DeploymentInfo } from "./billing-types";

// Public billing API (no auth required)
// Used for landing page pricing display
export const publicBillingApi = {
  getPricing: () =>
    publicRequest<PublicPricingResponse>("/api/v1/config/pricing"),

  getDeploymentInfo: () =>
    publicRequest<DeploymentInfo>("/api/v1/config/deployment"),
};
