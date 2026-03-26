# OAuth2 Token Management

This document describes the engine-managed OAuth2 token system for Gopherbot. It covers provider configuration, stored brain data, refresh behavior, and the extension API shape used by built-in and custom plugins.

Primary implementation files:
- `bot/oauth2.go`
- `bot/oauth2_types.go`
- `bot/oauth2_github_plugin.go`
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

The current built-in onboarding plugin is GitHub-specific:
- built-in Go plugin: `github-link`
- shipped config: `conf/plugins/github-link.yaml`

## Provider registry

Providers are configured in `robot.yaml` under `OAuth2Providers`.

Example:

```yaml
OAuth2Providers:
  github:
    DisplayName: GitHub
    ClientID: "Iv1..."
    ClientSecret: {{ decrypt "<encrypted github client secret>" }}
    DefaultScopes: [ "read:user", "repo", "workflow" ]
    TokenEndpointAuthMethod: client_id_only
    DeviceAuthorization:
      URL: https://github.com/login/device/code
      Headers:
        Accept: application/json
    Token:
      URL: https://github.com/login/oauth/access_token
      Headers:
        Accept: application/json
    UserInfo:
      URL: https://api.github.com/user
      Headers:
        Accept: application/vnd.github+json
```

Notes:
- Provider keys are normalized to lowercase during config load.
- `ClientSecret` may be omitted for providers and flows that only require `client_id`.
- Secrets should be stored encrypted with `gopherbot encrypt <secret>` and loaded with `{{ decrypt "..." }}`.
- `TokenEndpointAuthMethod` supports `client_secret_post`, `client_secret_basic`, `client_id_only`, and `none`.

## Stored brain schema

User link records are stored in the bot namespace, not plugin namespaces.

Datum key pattern:
- `bot:oauth2:v1:<provider-segment>:user:<sha1(username)>`

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

## Runtime token retrieval

`GetOAuth2Token(provider, user)` is the central retrieval helper.

Behavior:
1. load provider config from `currentCfg.oauth2Providers`
2. load the user link datum from the bot namespace
3. if the access token is present and not near expiry, return it
4. if expired, take a brain write lock on that datum and re-check state
5. if still expired, perform a refresh token grant when a refresh token exists
6. atomically write the new token state back before releasing the lock

Refresh behavior:
- refresh token rotation is handled automatically
- if the provider returns a new refresh token, it replaces the old one and increments `grant.rotation_counter`
- if the provider omits a new refresh token, the old refresh token is preserved
- if the provider omits `token_type` or `scope`, existing values are preserved
- provider errors such as `invalid_grant` mark the record as `reauth_required`

This locking model prevents two concurrent pipelines from racing the same refresh.

## Extension API

The engine exposes three public robot methods:

- `GetOAuth2Token(provider, user) (token string, ret RetVal)`
- `LinkOAuth2User(link *OAuth2LinkRequest) RetVal`
- `UnlinkOAuth2User(provider, user) RetVal`

`OAuth2LinkRequest` fields:
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
- `OAuth2ProviderNotFound`
- `OAuth2UserNotLinked`
- `OAuth2ReauthRequired`
- `OAuth2RefreshFailed`
- `OAuth2InvalidLinkRequest`
- `OAuth2ConfigError`

Design note:
- on failure, `GetOAuth2Token` returns `""` plus a machine-readable `RetVal`
- deeper detail is logged in engine logs and stored in the record status

## Built-in GitHub linker

`bot/oauth2_github_plugin.go` registers `github-link` as a built-in Go plugin.

Shipped commands:
- `link-github`
- `unlink-github`
- `github-whoami`

Flow:
1. start device authorization with the configured `github` provider
2. DM the user a verification URL and user code
3. poll the token endpoint until success, denial, or expiry
4. fetch `/user` from the GitHub API to confirm identity
5. persist the link with `LinkOAuth2User(...)`

This keeps the service-specific UX in the plugin while leaving token storage and refresh in the engine.

## Sample extension usage

Shipped examples use `GetOAuth2Token(...)` plus plain HTTP:
- `plugins/samples/github-tools.js`
- `jobs/samples/github-review-digest.js`

The intended model is:
- user links once with `link-github`
- plugin/job calls `GetOAuth2Token("github", bot.user)`
- plugin/job sends direct API requests with `Authorization: Bearer <token>`

No service-specific OAuth library is required for those extensions.
