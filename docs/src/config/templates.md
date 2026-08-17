# Config Templates

Gopherbot configuration files are Go text templates before they are parsed as YAML. This applies across the config tree, including `robot.yaml`, environment files, protocol files, provider files, plugin files, and job files.

Templates are most often used to:

- select environment-specific config
- read deployment environment variables
- include another config file
- reference shared variables and encrypted secrets
- keep custom robot config small while still supporting development, staging, and production differences

## Common Pattern

A scaffolded robot usually selects an environment file from `robot.yaml`:

```yaml
{{ $environment := env "GOPHER_ENVIRONMENT" | default "production" }}
{{ printf "environments/%s.yaml" $environment | .Include }}
```

With this pattern, `GOPHER_ENVIRONMENT=development` loads `conf/environments/development.yaml`; when the variable is unset, the robot loads `conf/environments/production.yaml`.

## Template Helpers

### env

`env` reads an environment variable.

```yaml
PrimaryProtocol: {{ env "GOPHER_PROTOCOL" | default "ssh" }}
```

If the variable is unset or empty, `env` returns an empty string.

### default

`default` returns a fallback value when the piped input is empty.

```yaml
LogLevel: {{ env "GOPHER_LOGLEVEL" | default "info" }}
```

### .Include

`.Include` includes another config file from the same config tree.

```yaml
{{ printf "environments/%s.yaml" $environment | .Include }}
```

When used from installed defaults, includes are resolved from the installed config tree. When used from custom config, includes are resolved from the custom robot config tree.

### secret

`secret` decrypts a named encrypted secret from the custom variables files.

```yaml
Password: '{{ secret "SMTP_PASSWORD" }}'
```

Secrets are loaded from:

- `custom/conf/variables/common.yaml`
- `custom/conf/variables/<environment>.yaml`

Environment-specific secrets override common secrets.

Example variables file:

```yaml
Secrets:
  SMTP_PASSWORD: "<encrypted-base64-value>"
```

### variable

`variable` reads a named plaintext value from the custom variables files.

```yaml
Value: '{{ variable "GITHUB_CLIENT_ID" }}'
```

Variables are loaded from:

- `custom/conf/variables/common.yaml`
- `custom/conf/variables/<environment>.yaml`

Environment-specific variables override common variables.

Example variables file:

```yaml
Variables:
  GITHUB_CLIENT_ID: "client-id"
```

### GetStartupMode

`GetStartupMode` returns the current startup mode.

Possible values:

- `demo`: no custom robot repository is configured
- `bootstrap`: a custom repository is configured but local config is not present yet
- `test-dev`: local development or integration-test config is being used
- `cli`: a CLI command is running instead of a full robot
- `production`: normal configured robot startup

Installed defaults use this helper to choose safe demo, bootstrap, test, and production behavior. Custom robot config usually does not need it; prefer `GOPHER_ENVIRONMENT` and environment files for normal robot differences.

### SetEnv

`SetEnv` overrides an environment variable during template expansion.

```yaml
{{ SetEnv "GOPHER_PROTOCOL" "ssh" }}
```

This is mainly useful in installed defaults because defaults load before custom config. Use it sparingly in custom robot config; ordinary variables and environment files are usually clearer.

### IsTestBuild

`IsTestBuild` is true when Gopherbot was compiled with the test build tag.

This helper is intended for Gopherbot's own test and default configuration. Most custom robots should not need it.

## Secrets and the Removed decrypt Helper

The old `decrypt` helper has been removed in v3. If a legacy config uses:

```yaml
Token: '{{ decrypt "..." }}'
```

move the encrypted value into a variables file:

```yaml
Secrets:
  TOKEN: "<encrypted-base64-value>"
```

then reference it with:

```yaml
Token: '{{ secret "TOKEN" }}'
```

This keeps encrypted values in one predictable place and makes it easier to separate development, staging, and production secrets.

## Quoting Template Values

YAML parsing happens after template expansion. Quote template expressions when the expanded value may contain punctuation, spaces, colons, braces, or other YAML-sensitive characters.

Recommended:

```yaml
Password: '{{ secret "SMTP_PASSWORD" }}'
```

Usually fine for simple values:

```yaml
PrimaryProtocol: {{ env "GOPHER_PROTOCOL" | default "ssh" }}
```

When in doubt, quote the template expression.

## Merge Timing

Each config file is expanded before it is merged. Installed defaults are expanded and loaded first. Custom robot config is expanded and merged afterward.

That means custom templates can use custom variables files, and custom scalar values replace installed defaults after expansion.
