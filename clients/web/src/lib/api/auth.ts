// Legacy authApi facade — now delegates to authConnect (Connect-RPC,
// binary wire). Kept as a re-export shim so downstream call sites
// migrate at their own pace; the underlying transport is already
// flipped to proto.auth.v1.AuthService.
//
// New code should import directly from @/lib/api/authConnect.

import * as authConnect from "@/lib/api/authConnect";

export const authApi = {
  register: async (data: {
    name?: string;
    email: string;
    username?: string;
    password: string;
  }) => {
    const username = data.username ?? data.email.split("@")[0];
    const session = await authConnect.register({
      email: data.email,
      username,
      password: data.password,
      name: data.name,
    });
    return {
      token: session.token,
      refresh_token: session.refresh_token,
      user: session.user,
      message: session.message,
    };
  },
  verifyEmail: async (token: string) => {
    const session = await authConnect.verifyEmail(token);
    return {
      token: session.token,
      refresh_token: session.refresh_token,
      user: session.user,
      message: session.message,
    };
  },
  resendVerification: async (email: string) => {
    return authConnect.resendVerification(email);
  },
  forgotPassword: async (email: string) => {
    return authConnect.forgotPassword(email);
  },
  resetPassword: async (token: string, password: string) => {
    return authConnect.resetPassword(token, password);
  },
};
