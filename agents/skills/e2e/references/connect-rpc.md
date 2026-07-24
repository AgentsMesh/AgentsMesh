# Connect-RPC Authentication Contract

AgentsMesh login uses the Connect unary endpoint:

```text
POST /proto.auth.v1.AuthService/Login
Content-Type: application/json
Connect-Protocol-Version: 1
```

Use protobuf JSON field names and encoding rules. In particular, field names
are lower camel case and 64-bit integers are represented as JSON strings.

The SSO discovery endpoint follows the same routing contract:

```text
POST /proto.sso.v1.SSOService/Discover
```

The development and production reverse proxies must route `/proto.` to the
Backend with higher priority than the frontend catch-all.

- A Backend credential failure should be a structured Connect error such as an
  unauthenticated response.
- An HTML 404 containing Next.js assets means the request reached the frontend;
  it does not prove the email or password is wrong.
- A JSON 404 from the Backend means the procedure or deployment version must be
  checked.

Do not use the removed `/api/v1/auth/login` route in new tests or diagnostics.
