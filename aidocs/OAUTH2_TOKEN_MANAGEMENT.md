# OAuth2 Identity Decisions

OAuth token storage and refresh are engine services. Service-specific plugins
own onboarding UX but must not duplicate refresh/storage policy.

Device Authorization Grant is the preferred onboarding flow because ChatOps,
SSH, and terminals can complete it with outbound HTTP only—no public callback
listener or redirect URI management.

## Security model

- Provider configuration and refresh credentials remain engine-private.
- Each identity API caller must have the provider's credential `ParameterSet`
  explicitly attached. Shipping an extension does not grant access.
- Links are keyed in the engine's bot namespace by provider and a hash of the
  canonical username, not in a plugin namespace or by transport ID.
- `GetIdentityCredential` returns a scoped usable credential, not the provider
  registry or raw parameter set.
- Onboarding plugins are explicit custom-robot opt-ins.

## Refresh model

The engine loads the link, returns a still-valid token, or locks that datum and
rechecks before refresh. Refresh-token rotation and status updates are written
atomically, preventing concurrent pipelines from racing one user's token.
Provider errors that require user action mark the link for reauthentication
rather than silently returning stale credentials.

The public compatibility surface is
`GetIdentityCredential`, `LinkOAuth2Identity`, and `UnlinkIdentity`; exact
types/return codes live in `robot/oauth2.go` and `robot/robot.go`.
