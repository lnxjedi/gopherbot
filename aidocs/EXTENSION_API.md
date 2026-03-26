# Extension API (Robot Methods)

This document catalogs the Gopherbot extension API across languages, with a focus on the methods extension authors call. It is intended as a reference for AI agents and contributors working on interpreter support.

Primary sources:
- Go interface: `robot/robot.go` (type `Robot`).
- External JSON API dispatch: `bot/http.go` (func `ServeHTTP` on type `handler`).
- Language libraries: `lib/gopherbot_v1.lua`, `lib/gopherbot_v1.js`, `lib/gopherbot_v1.sh`, `lib/gopherbot_v2.py`, `lib/gopherbot_v1.rb`.

## Canonical Robot interface (Go / Yaegi)

The authoritative API surface for compiled Go and Yaegi-based extensions is the `robot.Robot` interface in `robot/robot.go`.

### Identity, attributes, config
- `GetMessage()` – returns `*robot.Message` for the current pipeline.
- `GetTaskConfig(cfgptr interface{}) RetVal`
- `GetParameter(name string) string`
- `GetBotAttribute(a string) *AttrRet`
- `GetUserAttribute(u, a string) *AttrRet`
- `GetSenderAttribute(a string) *AttrRet`

### Messaging and formatting
- `Direct() Robot`, `Threaded() Robot`, `Fixed() Robot`, `MessageFormat(f MessageFormat) Robot`
- `SendChannelMessage(ch, msg string, v ...interface{}) RetVal`
- `SendChannelThreadMessage(ch, thr, msg string, v ...interface{}) RetVal`
- `SendUserChannelMessage(u, ch, msg string, v ...interface{}) RetVal`
- `SendProtocolUserChannelMessage(protocol, u, ch, msg string, v ...interface{}) RetVal`
- `SendUserChannelThreadMessage(u, ch, thr, msg string, v ...interface{}) RetVal`
- `SendUserMessage(u, msg string, v ...interface{}) RetVal`
- `Say(msg string, v ...interface{}) RetVal`, `SayThread(msg string, v ...interface{}) RetVal`
- `Reply(msg string, v ...interface{}) RetVal`, `ReplyThread(msg string, v ...interface{}) RetVal`

### Prompting
- `PromptForReply(regexID, prompt string, v ...interface{}) (string, RetVal)`
- `PromptThreadForReply(regexID, prompt string, v ...interface{}) (string, RetVal)`
- `PromptUserForReply(regexID, user, prompt string, v ...interface{}) (string, RetVal)`
- `PromptUserChannelForReply(regexID, user, channel, prompt string, v ...interface{}) (string, RetVal)`
- `PromptUserChannelThreadForReply(regexID, user, channel, thread, prompt string, v ...interface{}) (string, RetVal)`

Prompt timeout semantics:
- Default timeout: `45s`.
- Extended timeout: `42m` for `ssh`/`terminal` when the calling task is compiled Go or interpreter-backed (`.go`, `.lua`, `.js`, `.gsh`).
- On robot shutdown, in-progress prompt waits return `Interrupted` immediately.

### Memory (brain + ephemeral)
- `CheckoutDatum(key string, datum interface{}, rw bool) (locktoken string, exists bool, ret RetVal)`
- `CheckinDatum(key, locktoken string)`
- `UpdateDatum(key, locktoken string, datum interface{}) RetVal`
- `DeleteDatum(key string) RetVal`
- `Remember(key, value string, shared bool)`
- `RememberThread(key, value string, shared bool)`
- `RememberContext(context, value string)`
- `RememberContextThread(context, value string)`
- `Recall(key string, shared bool) string`
- `DeleteMemory(key string, shared bool)`

### OAuth2 token management
- `GetOAuth2Token(provider, user string) (string, RetVal)`
- `LinkOAuth2User(link *OAuth2LinkRequest) RetVal`
- `UnlinkOAuth2User(provider, user string) RetVal`

OAuth2 notes:
- `GetOAuth2Token` returns the raw bearer token string, not a full `Authorization` header.
- Token refresh/storage is engine-managed and uses internal provider config from `OAuth2Providers` in `robot.yaml`.
- Onboarding plugins should receive OAuth client credentials only through explicit per-plugin configuration such as `ParameterSets`, not by reading shared robot config through an API.
- `OAuth2LinkRequest` is defined in `robot/oauth2.go`.
- Return codes include `OAuth2ProviderNotFound`, `OAuth2UserNotLinked`, `OAuth2ReauthRequired`, `OAuth2RefreshFailed`, `OAuth2InvalidLinkRequest`, and `OAuth2ConfigError`.

Secret-access rule:
- `GetTaskConfig` and attached `ParameterSets` may contain secrets because the robot administrator explicitly scoped them to the calling extension.
- Generic unprivileged robot methods must not return shared secret-bearing configuration such as provider registries or other extensions' parameter sets.

### Pipeline control
- `Exclusive(tag string, queueTask bool) bool`
- `SpawnJob(name string, args ...string) RetVal`
- `AddTask(name string, args ...string) RetVal`
- `FinalTask(name string, args ...string) RetVal`
- `FailTask(name string, args ...string) RetVal`
- `AddJob(name string, args ...string) RetVal`
- `AddCommand(pluginName, command string) RetVal`
- `FinalCommand(pluginName, command string) RetVal`
- `FailCommand(pluginName, command string) RetVal`

Pipeline behavior notes:
- `AddJob` starts a child pipeline when the added job runs; it does not share parent `SetParameter` values by default.
- For job pipelines, `startPipeline` exposes origin metadata in environment/parameters such as `GOPHER_START_PROTOCOL`, `GOPHER_START_CHANNEL`, `GOPHER_START_THREAD_ID`, and `GOPHER_START_USER`.
- `GOPHER_START_MESSAGE_ID` is the connector-provided opaque message ID for the inbound event that started the job (when available); it may be empty for scheduled/init jobs.
- Job status output normally targets the configured job channel, but extension tasks can use `GOPHER_START_*` metadata to additionally notify the command-origin context when needed.

### Admin, logging, utilities
- `CheckAdmin() bool`
- `Subscribe() bool`, `Unsubscribe() bool`
- `Elevate(immediate bool) bool`
- `Log(l LogLevel, m string, v ...interface{}) bool`
- `RandomInt(n int) int`, `RandomString(s []string) string`, `Pause(s float64)`
- `Email(...)`, `EmailUser(...)`, `EmailAddress(...)` (see `robot/robot.go`)
- `RaisePriv(path string)` (Go-only)
- `SetParameter(name, value string) bool`
- `SetWorkingDirectory(path string) bool`

## External JSON API (HTTP)

External scripts (bash/python/ruby/etc.) call into the robot via JSON POSTs. The HTTP handler in `bot/http.go` dispatches on `FuncName` and `FuncArgs` and enforces the supported call set.

Supported `FuncName` values in `bot/http.go`:
- `CheckAdmin`, `Subscribe`, `Unsubscribe`
- `AddTask`, `AddJob`, `FinalTask`, `FailTask`, `SpawnJob`
- `AddCommand`, `FinalCommand`, `FailCommand`
- `SetParameter`, `SetWorkingDirectory`
- `Exclusive`, `Elevate`
- `GetOAuth2Token`, `LinkOAuth2User`, `UnlinkOAuth2User`
- `CheckoutDatum`, `CheckinDatum`, `UpdateDatum`, `DeleteDatum`
- `Remember`, `RememberThread`, `Recall`, `DeleteMemory`
- `GetParameter`, `GetTaskConfig`
- `GetSenderAttribute`, `GetBotAttribute`, `GetUserAttribute`
- `Log`
- `SendChannelThreadMessage`, `SendUserChannelThreadMessage`, `SendProtocolUserChannelMessage`, `SendUserMessage`
- `PromptUserChannelThreadForReply`

Notes:
- `bot/http.go` explicitly notes that `Say`, `Reply`, and the user-level prompt helpers are implemented in the language libraries, not the HTTP handler.
- External libraries use `GOPHER_HTTP_POST` and `X-Caller-ID` headers for requests (`lib/gopherbot_v1.sh`, `lib/gopherbot_v2.py`).

## Built-in interpreter libraries (Lua / JavaScript / Gopherbot shell)

Lua and JavaScript run in-process but use the same logical API surface via their libraries:

- Lua: `lib/gopherbot_v1.lua` defines `Robot:new()` and exposes the primary methods.
- JavaScript: `lib/gopherbot_v1.js` defines `new Robot()` and exposes the primary methods.

Both wrappers use the `GBOT` global injected by the interpreter modules (`lib/gopherbot_v1.lua`, `lib/gopherbot_v1.js`). They mirror most of the `robot.Robot` interface and are the canonical method list for Lua/JS extensions.

OAuth2 parity note:
- Lua and JavaScript both expose `GetOAuth2Token`, `LinkOAuth2User`, and `UnlinkOAuth2User`.

Gopherbot shell uses `modules/gsh/assets/gopherbot_v1.gsh` as a compatibility shim, but the primary interface is builtin shell commands rather than a loaded language object:

- Robot methods are exposed as shell builtins (`say`, `Reply`, `PromptForReply`, `CheckAdmin`, `AddTask`, `GetTaskConfig`, etc.).
- Common utility commands are also builtin (`base64`, `cat`, `cp`, `find`, `grep`, `jq`, `ls`, `mktemp`, `mv`, `rm`, `sort`, `tar`, `touch`, `tr`, `uniq`, `wc`, `xargs`, and related helpers).
- `say` / `Say` style variants are equivalent because command lookup normalizes case plus `-` / `_`.
- `.gsh` does not use `bot/http.go`; Robot methods traverse the internal pipeline RPC robot bridge instead.

## External interpreter libraries (Bash / Python / Ruby)

External interpreters call the HTTP API and wrap it in language-appropriate helpers:

- Bash: `lib/gopherbot_v1.sh` exports functions like `Say`, `Reply`, `Remember`, `PromptForReply`, `AddTask`, and more; it uses curl to post JSON to `GOPHER_HTTP_POST`.
- Python 3: `lib/gopherbot_v2.py` defines `class Robot` with the same core methods, plus `Subscribe`, `Unsubscribe`, and `SetWorkingDirectory`.
- Ruby: `lib/gopherbot_v1.rb` defines `class Robot` (via `BaseBot`) with the same core methods, plus `Subscribe`, `Unsubscribe`, and `SetWorkingDirectory`.
- Bash, Python, Ruby, Julia, and compatibility JS/Lua libraries also expose the OAuth2 methods above.

## Parity notes and known gaps

- `Subscribe` / `Unsubscribe` are now part of the canonical Go interface (`robot/robot.go`) and are exercised for external yaegi plugins via `test/go_full_test.go` + `plugins/test/gofull.go`.
- `SetWorkingDirectory` exists in the Go interface and external libraries (`lib/gopherbot_v1.sh`, `lib/gopherbot_v2.py`, `lib/gopherbot_v1.rb`), but it is not present in the Lua/JS wrappers as of `lib/gopherbot_v1.lua` / `lib/gopherbot_v1.js`.
- `.gsh` implements `SetWorkingDirectory` plus a BusyBox-style builtin utility surface in-process inside the child interpreter.
- `RaisePriv` is Go-only (`robot/robot.go`); there is no wrapper in external language libraries.
- Yaegi caveat: interpreted Go plugins can diverge from compiled Go when values cross reflective boundaries. A focused local repro in `modules/yaegi-dynamic-go/yaegi_dynamic_test.go` shows that a helper chain returning a mixed multi-value tuple such as `(conversationState, []conversationExchange)` can panic under `RunPluginHandler` with `reflect.Set ... not assignable`, even though the same pattern succeeds in compiled Go.
- For external Go plugins running under Yaegi, prefer returning a single wrapper struct when state must carry multiple logically-related values across helper boundaries. The `plugins/go-openai-fallback` compaction path now uses `compactionResult{State, Older}` for this reason.
- As of March 11, 2026, no exact upstream Yaegi issue was identified for this specific panic. The behavior is consistent with Yaegi's documented limitation that `reflect` type representation can differ between compiled and interpreted execution.

## Related docs

- `aidocs/INTERPRETERS.md` – execution model and interpreter categories
- `aidocs/EXTENSION_SURFACES.md` – where extensions live and how they are wired
