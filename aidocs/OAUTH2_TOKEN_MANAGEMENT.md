# OAuth2 Token Management

This document describes the engine-managed OAuth2 token system for Gopherbot. It covers provider configuration, stored brain data, refresh behavior, and the extension API shape used by built-in and custom plugins.

Primary implementation files:
- `bot/oauth2.go`
- `bot/oauth2_types.go`
- `plugins/go-github-link/github_link.go`
- `robot/oauth2.go`
- `robot/robot.go`

## Goals

- Keep OAuth2 refresh/storage logic in engine code, not duplicated across plugins.
- Let service-specific onboarding plugins stay thin and service UX focused.
- Expose a language-agnostic robot API so simple HTTP-based integrations can run in JavaScript, Lua, shell, Python, Ruby, Go, and Yaegi without vendor OAuth libraries.
- Preserve Gopherbot's outbound-only operating model: no inbound callback listener is required.

## Supported onboarding model

The first-class onboarding flow is OAuth2 Device Authorization Grant (RFC 8628).

Why device flow is the default:
- fits chatops, CLI, SSH, and terminal workflows
- requires only outbound HTTP
- avoids redirect URI management and public callback endpoints
- allows the robot to use its own client credentials during token exchange

The current shipped onboarding plugin is GitHub-specific:
- shipped external Go plugin: `github-link`
- shipped config: `conf/plugins/github-link.yaml`

## Provider registry

Providers are configured in `robot.yaml` under `IdentityProviders`.

The provider registry is internal engine config for refresh behavior. It is not exposed to plugins.

Example:

```yaml
ParameterSets:
  github_oauth:
    Parameters:
    - Name: CLIENT_ID
      Value: "Iv1..."
    - Name: CLIENT_SECRET
      Value: {{ secret "GITHUB_CLIENT_SECRET" | printf "%q" }}

IdentityProviders:
  github:
    Type: oauth2
    CredentialParameterSet: github_oauth
    OAuth2:
      TokenEndpointAuthMethod: client_secret_post
      Token:
        URL: https://github.com/login/oauth/access_token
        Headers:
          Accept: application/json

ExternalPlugins:
  "github-link":
    Path: plugins/go-github-link/github_link.go
    ParameterSets:
    - github_oauth
```

Notes:
- Provider keys are normalized to lowercase during config load.
- `CredentialParameterSet` points to a standard parameter set containing `CLIENT_ID` and `CLIENT_SECRET`.
- Secrets should be stored encrypted with `gopherbot encrypt <secret>` under custom `conf/variables/*.yaml` `Secrets`, then loaded with `{{ secret "NAME" }}`.
- `TokenEndpointAuthMethod` supports `client_secret_post`, `client_secret_basic`, `client_id_only`, and `none`.
- Any extension that calls `GetIdentityCredential`, `LinkOAuth2Identity`, or `UnlinkIdentity` must have that same parameter set attached through its own `ParameterSets` config.
- Shipped credentialed extensions are opt-in for custom robots; they should not be assumed active in the default robot.

## Stored brain schema

User link records are stored in the bot namespace, not plugin namespaces.

Datum key pattern:
- `bot:identity:v1:<provider-segment>:user:<sha1(username)>`

Storage notes:
- the provider segment is sanitized for brain/file safety
- the username portion is hashed so raw usernames are not embedded in datum keys
- the stored payload still records the canonical username for lookup/debugging

Stored JSON shape:

```json
{
  "schema_version": 1,
  "provider_key": "github",
  "username": "alice",
  "subject": {
    "id": "12345",
    "login": "alice-gh",
    "name": "Alice Example",
    "email": "alice@example.com"
  },
  "token": {
    "access_token": "....",
    "refresh_token": "....",
    "token_type": "Bearer",
    "scope": ["repo", "workflow"],
    "expires_at": "2026-03-25T16:00:00Z",
    "refresh_expires_at": null
  },
  "grant": {
    "type": "device_code",
    "linked_at": "2026-03-25T15:00:00Z",
    "last_refresh_at": "2026-03-25T15:30:00Z",
    "rotation_counter": 1
  },
  "status": {
    "reauth_required": false,
    "last_error_code": 0,
    "last_error": "",
    "last_error_at": null
  }
}
```

## Runtime credential retrieval

`GetIdentityCredential(provider, user)` is the central retrieval helper for user-linked identity providers.

Behavior:
1. load provider config from `currentCfg.identityProviders`
2. verify the calling task/plugin/job has the provider's `CredentialParameterSet` attached
3. resolve client credentials internally from the provider's `CredentialParameterSet`
4. load the user link datum from the bot namespace
5. if the access token is present and not near expiry, return a structured credential envelope
6. if expired, take a brain write lock on that datum and re-check state
7. if still expired, perform a refresh token grant when a refresh token exists
8. atomically write the new token state back before releasing the lock

Refresh behavior:
- refresh token rotation is handled automatically
- if the provider returns a new refresh token, it replaces the old one and increments `grant.rotation_counter`
- if the provider omits a new refresh token, the old refresh token is preserved
- if the provider omits `token_type` or `scope`, existing values are preserved
- provider errors such as `invalid_grant` mark the record as `reauth_required`

This locking model prevents two concurrent pipelines from racing the same refresh.

## Extension API

The engine exposes three public robot methods:

- `GetIdentityCredential(provider, user) (*IdentityCredential, RetVal)`
- `LinkOAuth2Identity(link *OAuth2IdentityLinkRequest) RetVal`
- `UnlinkIdentity(provider, user) RetVal`

`GetIdentityCredential` returns an `IdentityCredential` object with fields such as:
- `Type`
- `Value`
- `Scheme`
- `HeaderName`
- `HeaderValue`
- `ExpiresAt`

`OAuth2IdentityLinkRequest` fields:
- `Provider`
- `User`
- `AccessToken`
- `RefreshToken`
- `TokenType`
- `Scope`
- `ExpiresIn`
- `ExpiresAt`
- `RefreshExpiresIn`
- `RefreshExpiresAt`
- `GrantType`
- `SubjectID`
- `SubjectLogin`
- `SubjectName`
- `SubjectEmail`

Return codes:
- `IdentityProviderNotFound`
- `IdentityNotLinked`
- `IdentityReauthRequired`
- `IdentityRefreshFailed`
- `IdentityInvalidLinkRequest`
- `IdentityConfigError`

Design note:
- on failure, `GetIdentityCredential` returns `nil` plus a machine-readable `RetVal`
- deeper detail is logged in engine logs and stored in the record status
- missing caller `ParameterSet` attachment is reported as `IdentityConfigError` with engine logs naming the provider, caller, required `CredentialParameterSet`, and attached `ParameterSets`

## Shipped GitHub linker

`plugins/go-github-link/github_link.go` implements `github-link` as a shipped external Go plugin for custom robots.

Shipped commands:
- `link-github`
- `unlink-github`
- `github-whoami`

Flow:
1. read `CLIENT_ID` and `CLIENT_SECRET` from the plugin's explicitly attached `ParameterSets`
2. start device authorization against GitHub
3. DM the user a verification URL and user code
4. poll the token endpoint until success, denial, or expiry
5. fetch `/user` from the GitHub API to confirm identity
6. persist the link with `LinkOAuth2Identity(...)`

This keeps the service-specific UX in the plugin while leaving token storage and refresh in the engine.

Security boundary:
- onboarding plugins get secrets only because the robot owner explicitly attached the credential parameter set to that plugin
- any extension calling the identity methods must also have that same provider credential `ParameterSet` attached
- `GetIdentityCredential` uses the provider registry internally for refresh, but plugins cannot read the registry back out through a robot method

## Sample extension usage

Shipped examples use `GetIdentityCredential(...)` plus plain HTTP:
- `plugins/samples/github-tools.js`
- `jobs/samples/github-review-digest.js`

The intended model is:
- user links once with `link-github`
- plugin/job with `github_oauth` attached calls `GetIdentityCredential("github", bot.user)`
- plugin/job sends direct API requests with `credential.header_value` or `credential.value`

No service-specific OAuth library is required for those extensions.
