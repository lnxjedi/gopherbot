# Security Proof: Integration Coverage for Engine Security Controls

## Status Update

The prior concern about missing-authorizer detection has been addressed in this slice:

- Load-time validation now fails fast for plugins that require authorization but have no effective authorizer.
- Runtime validation is still retained as defense in depth.

Load-time validation is implemented in `bot/taskconf.go`, and runtime validation remains in `bot/authorize.go`.

Privsep hardening status in this slice:

- Startup now panics with an explicit tamper message if a setuid-`nobody` `gopherbot` binary has unsafe mode bits or owner mismatch.
- The binary encryption key file is validated (no symlink / regular file) before read and permissions are enforced to owner-only (`0600`).
- Child-runner process launches now explicitly strip high-sensitivity `GOPHER_*` secrets (`GOPHER_ENCRYPTION_KEY`, `GOPHER_DEPLOY_KEY`, `GOPHER_HOST_KEYS`) from inherited environment.

## Privsep Caveats and Residual Risk

1. Environment inheritance is still broad by design for non-sensitive variables.

We now strip high-risk secrets from child runner environments, but child processes still inherit most parent environment entries. This is acceptable under the current model (operators should not place secrets in generic process env), but it remains a defense-in-depth consideration.

2. Robot code/config readability tradeoff is intentional.

Unprivileged plugin execution needs read/execute access to plugin files and libraries, so we cannot make all of `custom/` unreadable to the unprivileged execution UID. This means privsep protects process privileges, not confidentiality of plugin source/config from code running as the unprivileged plugin user.

3. Untrusted plugins remain out of scope for trust guarantees.

Privsep reduces blast radius for privilege escalation, but does not make untrusted extension code safe in an absolute sense. Operators should still treat plugin provenance and review as a core security boundary.

## Scope

This proof covers engine security controls exercised through:

- Built-in interpreter full suites:
  - JavaScript: `test/js_full_test.go` (`TestJSFullSecurity`)
  - Lua: `test/lua_full_test.go` (`TestLuaFullSecurity`)
  - Go/Yaegi: `test/go_full_test.go` (`TestGoFullSecurity`)
- External JSON/HTTP interpreter path:
  - Python3: `test/python_security_integration_test.go` (`TestPythonSecurity`)

Security test plugin wiring:

- JS full suite: `test/jsfull/conf/robot.yaml`
- Lua full suite: `test/luafull/conf/robot.yaml`
- Go full suite: `test/gofull/conf/robot.yaml`
- Python security suite: `test/membrain/conf/robot.yaml`

## Control-by-Control Proof Map

### 1) `AdminCommands`

Config snippet (`test/jsfull/conf/plugins/jssec.yaml`):

```yaml
AdminCommands:
- secadmincmd
```

Plugin snippet (`plugins/test/jsfull.js`):

```javascript
case 'secadmincmd': {
  const bot = new Robot();
  bot.Say(`SECURITY CHECK: ${cmd}`);
  return task.Normal;
}
```

Integration test snippet (`test/js_full_test.go`):

```go
{aliceID, general, ";js-sec-admincmd", ... "SECURITY CHECK: secadmincmd" ...},
{bobID, general, ";js-sec-admincmd", ... "only available to bot administrators" ...},
```

### 2) `RequireAdmin`

Config snippet (`test/jsfull/conf/plugins/jssec-adminonly.yaml`):

```yaml
RequireAdmin: true
CommandMatchers:
- Regex: (?i:js-sec-adminonly)
  Command: secadminonly
```

Test snippet (`test/js_full_test.go`):

```go
{aliceID, general, ";js-sec-adminonly", ... "SECURITY CHECK: secadminonly" ...},
{bobID, general, ";js-sec-adminonly", ... "No command matched in channel.*" ...},
```

### 3) `Users` allowlist

Config snippet (`test/jsfull/conf/plugins/jssec-users.yaml`):

```yaml
Users:
- alice
CommandMatchers:
- Regex: (?i:js-sec-usersonly)
  Command: secusersonly
```

Test snippet (`test/js_full_test.go`):

```go
{aliceID, general, ";js-sec-usersonly", ... "SECURITY CHECK: secusersonly" ...},
{bobID, general, ";js-sec-usersonly", ... "No command matched in channel.*" ...},
```

### 4) `AuthorizedCommands` + `AuthRequire` + global default authorizer

Config snippets:

- `test/jsfull/conf/robot.yaml`:

```yaml
DefaultAuthorizer: groups
```

- `test/jsfull/conf/plugins/jssec.yaml`:

```yaml
AuthorizedCommands:
- secauthz
AuthRequire: Helpdesk
```

Test snippet (`test/js_full_test.go`):

```go
{bobID, general, ";js-sec-authz", ... "SECURITY CHECK: secauthz" ...},
{davidID, general, ";js-sec-authz", ... "Sorry, you're not authorized for that command" ...},
```

### 5) `AuthorizeAllCommands`

Config snippet (`test/jsfull/conf/plugins/jssec-authall.yaml`):

```yaml
AuthorizeAllCommands: true
AuthRequire: Helpdesk
CommandMatchers:
- Regex: (?i:js-sec-authall)
  Command: secauthall
```

Test snippet (`test/js_full_test.go`):

```go
{bobID, general, ";js-sec-authall", ... "SECURITY CHECK: secauthall" ...},
{davidID, general, ";js-sec-authall", ... "Sorry, you're not authorized for that command" ...},
```

### 6) `ElevatedCommands` + `ElevateImmediateCommands` + default elevator

Config snippets:

- `test/jsfull/conf/robot.yaml`:

```yaml
DefaultElevator: builtin-totp
```

- `test/jsfull/conf/plugins/jssec.yaml`:

```yaml
ElevatedCommands:
- secelevated
ElevateImmediateCommands:
- secimmediate
```

Test snippet (`test/js_full_test.go`):

```go
{aliceID, general, ";js-sec-elevated", ... "This command requires.*elevation.*TOTP code.*" ...},
{aliceID, general, ";js-sec-immediate", ... "This command requires immediate elevation.*TOTP code.*" ...},
```

### 7) `AllowedHiddenCommands` and hidden-command deny path

Config snippet (`test/jsfull/conf/plugins/jssec.yaml`):

```yaml
AllowedHiddenCommands:
- sechiddenok
```

Test snippet (`test/js_full_test.go`):

```go
{aliceID, general, "/;js-sec-hidden-ok", ... "\\(SECURITY CHECK: sechiddenok\\)" ...},
{aliceID, general, "/;js-sec-hidden-denied", ... "cannot be run as a hidden command" ...},
```

### 8) Missing authorizer misconfiguration (load-time fail-fast)

Misconfig setup (`test/membrain/conf/plugins/pysec-misconfig.yaml`):

```yaml
Authorizer: ""
AuthorizedCommands:
- secmisconfig
```

Outcome proof (`test/python_security_integration_test.go`):

```go
{aliceID, general, ";python-sec-auth-misconfig", ... "No command matched in channel.*" ...},
```

This now proves load-time disabling for the misconfigured plugin (fail-fast), while runtime guards still remain in engine code.

## Cross-Interpreter Coverage Matrix

| Security control | JS | Lua | Go | Python3 |
| --- | --- | --- | --- | --- |
| `AdminCommands` | `TestJSFullSecurity` | `TestLuaFullSecurity` | `TestGoFullSecurity` | `TestPythonSecurity` |
| `RequireAdmin` | `TestJSFullSecurity` | `TestLuaFullSecurity` | `TestGoFullSecurity` | `TestPythonSecurity` |
| `Users` allowlist | `TestJSFullSecurity` | `TestLuaFullSecurity` | `TestGoFullSecurity` | `TestPythonSecurity` |
| `AuthorizedCommands` + group auth | `TestJSFullSecurity` | `TestLuaFullSecurity` | `TestGoFullSecurity` | `TestPythonSecurity` |
| `AuthorizeAllCommands` | `TestJSFullSecurity` | `TestLuaFullSecurity` | `TestGoFullSecurity` | `TestPythonSecurity` |
| `ElevatedCommands` | `TestJSFullSecurity` | `TestLuaFullSecurity` | `TestGoFullSecurity` | `TestPythonSecurity` |
| `ElevateImmediateCommands` | `TestJSFullSecurity` | `TestLuaFullSecurity` | `TestGoFullSecurity` | `TestPythonSecurity` |
| `AllowedHiddenCommands` + hidden deny | `TestJSFullSecurity` | `TestLuaFullSecurity` | `TestGoFullSecurity` | `TestPythonSecurity` |
| Missing authorizer misconfig path | N/A | N/A | N/A | `TestPythonSecurity` |

## Re-run Commands

```bash
TEST=JSFull make integration
TEST=LuaFull make integration
TEST=GoFull make integration
go test -run TestPythonSecurity -v --tags 'test integration netgo osusergo static_build' -mod readonly -race ./test
```

Captured pass lines from this run:

```text
--- PASS: TestJSFullSecurity
--- PASS: TestLuaFullSecurity
--- PASS: TestGoFullSecurity
--- PASS: TestPythonSecurity
PASS
```
