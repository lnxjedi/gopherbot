# Environment-Scoped Secrets And Variables Design

Status: current v3 behavior and implementation design notes.

Primary roadmap source: root `GOALS_v3.md`, section "Consolidation of Secrets and Variables".

## Summary

The roadmap design is suitable for v3 with one important boundary: `secret` and `variable` should be configuration-template helpers only. They must not become generic Robot API methods, and they must not expose a global secret registry to extensions at runtime.

The design makes `GOPHER_ENVIRONMENT` behave more like conventional deployment environments:

- `common.yaml` carries values shared by all environments.
- `<environment>.yaml` carries environment-specific overrides.
- encrypted secrets and plaintext variables live in a dedicated custom-robot
  variables layer.
- normal robot, protocol, provider, plugin, job, and task config references names instead of embedding ciphertext.
- the old config-template `decrypt` function is removed and any remaining use
  fails with a migration error so new config cannot scatter inline encrypted
  values across arbitrary files.

This keeps configuration easier to review while preserving the existing extension secret boundary: extensions only receive secrets through explicitly attached task config, `ParameterSets`, or their own brain namespace.

## Goals

- Give each `GOPHER_ENVIRONMENT` a clean secret/variable namespace.
- Keep encrypted secret declarations centralized under `conf/variables/`.
- Keep environment selection deterministic and tied to existing startup encryption key selection.
- Preserve normal installed-default/custom-override config layering outside the
  variables layer.
- Preserve v3's explicit secret scoping for extensions.
- Make config migration fail fast when old inline `{{ decrypt "..." }}` references remain.

## Non-Goals

- Do not remove the CLI `gopherbot decrypt` command; only remove the template function named `decrypt`.
- Do not remove the privileged Robot API method `EncryptSecret`; update its docs so it produces ciphertext suitable for `Secrets:` values.
- Do not add runtime Robot methods such as `GetSecret`, `GetVariable`, `ListSecrets`, or `ListVariables`.
- Do not allow extensions to inspect the variables registry.
- Do not make variables files templated. They are data files loaded before template expansion.

## Files And Layout

Variables files live only in custom robot configuration:

```text
conf/variables/common.yaml
conf/variables/<environment>.yaml
```

There is no installed/default `conf/variables/` tree. Unlike other config YAML,
variables are not layered from `$GOPHER_INSTALLDIR/conf`; they are entirely
robot-owned deployment data.

A configured robot may load up to two files:

```text
$GOPHER_CONFIGDIR/conf/variables/common.yaml
$GOPHER_CONFIGDIR/conf/variables/<environment>.yaml
```

Missing variables files are allowed. Missing referenced keys are not allowed.

Expected shape:

```yaml
Secrets:
  SLACK_TOKEN: "<base64 ciphertext>"
  WEATHER_API_KEY: "<base64 ciphertext>"
Variables:
  OUTPUT_CHANNEL: "test-jobs"
  ROBOT_NAME: "Clu"
```

Only top-level `Secrets` and `Variables` keys are valid. Values should be strings. A `null` value deletes an inherited key so an environment can intentionally remove a common value.

## Effective Environment

The variables loader should use the same effective environment name as custom robot configuration:

- read `GOPHER_ENVIRONMENT`
- use the configured v3 default if unset
- validate it as a single path segment before deriving `conf/variables/<environment>.yaml`

For the current skeleton behavior, that means production is the template default and onboarding writes `GOPHER_ENVIRONMENT=development` into `.env` for local-first starts. If the v3 roadmap changes the default to development, update `robot.skel/conf/robot.yaml`, `aidocs/STARTUP_FLOW.md`, and `UPGRADING-v3.md` together.

The environment name must not contain `/`, `\`, path traversal, or empty segments.

## Load Order

Startup already initializes encryption before pre-connect config load. The variables layer should be loaded after encryption initialization and before any templated config file expansion in a given config load.

Recommended precedence:

1. custom `conf/variables/common.yaml`
2. custom `conf/variables/<environment>.yaml`

Later files override earlier files. Environment files override `common.yaml`.

This variables precedence is intentionally smaller than normal config-file merge
semantics. Active installed defaults must not require variables, because demo
and bootstrap startup can run without a custom robot repository. Inert shipped
samples may demonstrate `secret` or `variable` references, but those references
are not usable until a custom robot repository supplies the matching variables
files. The engine distribution must not carry default variable values.

## Template Functions

Add these functions to the config template FuncMap:

```gotemplate
{{ secret "SLACK_TOKEN" }}
{{ variable "OUTPUT_CHANNEL" }}
```

Semantics:

- `secret` looks up an active merged `Secrets` entry, base64-decodes it, decrypts it with the active robot encryption key, and returns plaintext.
- `variable` looks up an active merged `Variables` entry and returns its string value.
- missing keys fail template expansion.
- invalid ciphertext, base64 errors, and decryption failures fail template expansion.
- lookup names are case-sensitive; use uppercase underscore names by convention.
- returned values are raw strings, not automatically YAML-quoted.

Because returned values are raw strings, config authors should quote YAML-sensitive values explicitly:

```yaml
ProtocolConfig:
  SlackToken: {{ secret "SLACK_TOKEN" | printf "%q" }}
  DefaultChannel: {{ variable "OUTPUT_CHANNEL" | printf "%q" }}
  HostKey: {{ secret "SSH_HOST_KEY" | printf "%q" }}
```

For simple tokens that are known YAML-safe, unquoted use remains possible, but quoted use should be preferred in examples.

## Removing `decrypt`

Remove `decrypt` from the template FuncMap in `bot/config_load.go`.

Any v3 config containing:

```gotemplate
{{ decrypt "<ciphertext>" }}
```

must fail during `gopherbot validate` or startup template parsing. That is
intentional. The error should be explicit enough for an operator to migrate
without reading code, for example:

```text
template function "decrypt" was removed in v3; move this encrypted value to
custom/conf/variables/common.yaml or custom/conf/variables/<environment>.yaml
under Secrets and reference it with {{ secret "NAME" }}
```

The CLI decrypt command remains useful for operator recovery and debugging:

```bash
gopherbot decrypt <base64>
gopherbot decrypt -file <path>
```

Only config-template use of `decrypt` is removed.

## Secret Boundary

The variables registry is an engine-owned configuration input, not an extension surface.

Allowed flow:

1. config templates call `secret` or `variable`
2. resolved values become ordinary parsed config
3. extensions receive values only if the administrator explicitly placed them in that extension's task config, `Parameters`, `ParameterSets`, provider credential set, or another already-scoped config location

Disallowed flow:

- plugin calls a Robot API to read global variable or secret maps
- plugin lists known secret names
- plugin reads provider registries or other broad secret-bearing config
- connector-local code owns shared policy for secret selection

This preserves the existing rule from `aidocs/EXECUTION_SECURITY_MODEL.md`: generic unprivileged robot methods must not disclose shared secret-bearing configuration.

## CLI Changes

Keep `gopherbot encrypt` as the low-level way to create ciphertext for `Secrets:` values.

Update examples and helper output so they point at variables files instead of inline decrypt templates. For example, `gentotp` should stop printing `{{ decrypt "..." }}` snippets and should print a `Secrets:` entry or a migration hint.

Add a `genkey` command for environment-specific binary key management:

```bash
gopherbot genkey
gopherbot genkey -environment development
gopherbot genkey -environment staging -write
```

Recommended behavior:

- use the current `GOPHER_ENCRYPTION_KEY` as the wrapping key
- generate a fresh random 32-byte robot data key
- encrypt and base64-encode it using the same binary-key format as `binary-encrypted-key`
- print to stdout by default
- with `-write`, write atomically to `binary-encrypted-key` for production or `binary-encrypted-key.<environment>` for named non-production environments
- write files with the same permission policy as existing binary key files
- avoid starting normal robot runtime, brain, connectors, or plugins

This command should make it easy to create a separate development or staging secret domain while retaining the existing `GOPHER_ENCRYPTION_KEY` deployment workflow.

## Migration

Before:

```yaml
ParameterSets:
  weather:
    Parameters:
    - Name: WEATHER_API_KEY
      Value: {{ decrypt "<ciphertext>" }}
```

After:

```yaml
# conf/variables/common.yaml
Secrets:
  WEATHER_API_KEY: "<ciphertext>"
```

```yaml
# conf/robot.yaml or conf/plugins/weather.yaml
ParameterSets:
  weather:
    Parameters:
    - Name: WEATHER_API_KEY
      Value: {{ secret "WEATHER_API_KEY" | printf "%q" }}
```

Environment override:

```yaml
# conf/variables/development.yaml
Secrets:
  WEATHER_API_KEY: "<development ciphertext>"
Variables:
  OUTPUT_CHANNEL: "dev-jobs"
```

```yaml
DefaultJobChannel: {{ variable "OUTPUT_CHANNEL" | printf "%q" }}
```

For environments with separate `binary-encrypted-key.<environment>` files, every active merged secret for that environment must be encrypted with that environment's active data key. If a common secret cannot decrypt under all environments, move it into the relevant environment files.

## Implementation Notes

Use a per-config-load context rather than a package-global mutable registry where practical:

- compute the effective environment once per `loadConfig(...)` call
- load and merge raw variables files without template expansion
- validate variable file schema and environment path safety
- make the merged registry available to all template expansion performed during that config load
- only update active runtime config after normal config parsing succeeds

Avoid loading variables through `getConfigFile(...)`; that path expands templates
and participates in installed/custom config merge semantics meant for runtime
config files. The variables loader should read only from `configPath`, never
from `installPath`.

Function signatures should return `(string, error)` so `text/template` aborts on missing keys or bad ciphertext instead of logging and returning an empty string.

Config reload should reload variables as part of the same config-load attempt. A failed reload must leave the currently running configuration intact.

## Implementation Files

- `bot/config_load.go`: variables loader, template FuncMap, removal of `decrypt`
- `bot/conf.go`: variables loading before config expansion
- `bot/cli_commands.go`: `genkey`, output/help text updates
- `bot/bot_process.go`: reuse existing binary key path and permission helpers where practical
- `robot/robot.go` and language docs: update `EncryptSecret` wording, keep API compatibility
- `robot.skel/conf/`: add variables files and replace skeleton inline decrypt references
- `lib/newrobotflow/onboarding.go`: write encrypted generated SSH host key into variables files instead of protocol config
- `conf/*.yaml.sample` and docs: replace examples that teach inline `decrypt`

## Validation Plan

Focused tests:

- variables precedence: custom common, then custom env
- environment-specific override beats common
- `null` deletes an inherited key
- installed `conf/variables/` files are ignored even if present
- missing `secret` and `variable` references fail config expansion
- invalid environment names are rejected
- invalid variables schema fails config load
- invalid base64/ciphertext fails config load
- removed `decrypt` template function fails old config
- config reload failure leaves prior runtime config active
- `genkey` produces a decryptable binary key and writes the right filename when requested

Broader verification:

- `go test ./bot`
- focused onboarding tests around generated robot skeleton content
- `make test`
- process-backed integration suite through `gopherbot-mcp` when startup/onboarding behavior changes
- `helpers/check-docs-hygiene.sh` for every docs/config migration change

## Documentation Anchors

Behavior is also documented in:

- `aidocs/STARTUP_FLOW.md`
- `aidocs/EXECUTION_SECURITY_MODEL.md`
- `aidocs/EXTENSION_API.md`
- `aidocs/OAUTH2_TOKEN_MANAGEMENT.md`
- root `UPGRADING-v3.md`
- relevant examples in `conf/` and `robot.skel/`
