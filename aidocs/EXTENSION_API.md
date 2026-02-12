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

### Memory (brain + ephemeral)
- `CheckoutDatum(key string, datum interface{}, rw bool) (locktoken string, exists bool, ret RetVal)`
- `CheckinDatum(key, locktoken string)`
- `UpdateDatum(key, locktoken string, datum interface{}) RetVal`
- `Remember(key, value string, shared bool)`
- `RememberThread(key, value string, shared bool)`
- `RememberContext(context, value string)`
- `RememberContextThread(context, value string)`
- `Recall(key string, shared bool) string`

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

### Admin, logging, utilities
- `CheckAdmin() bool`
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
- `CheckoutDatum`, `CheckinDatum`, `UpdateDatum`
- `Remember`, `RememberThread`, `Recall`
- `GetParameter`, `GetTaskConfig`
- `GetSenderAttribute`, `GetBotAttribute`, `GetUserAttribute`
- `Log`
- `SendChannelThreadMessage`, `SendUserChannelThreadMessage`, `SendProtocolUserChannelMessage`, `SendUserMessage`
- `PromptUserChannelThreadForReply`

Notes:
- `bot/http.go` explicitly notes that `Say`, `Reply`, and the user-level prompt helpers are implemented in the language libraries, not the HTTP handler.
- External libraries use `GOPHER_HTTP_POST` and `X-Caller-ID` headers for requests (`lib/gopherbot_v1.sh`, `lib/gopherbot_v2.py`).

## Built-in interpreter libraries (Lua / JavaScript)

Lua and JavaScript run in-process but use the same logical API surface via their libraries:

- Lua: `lib/gopherbot_v1.lua` defines `Robot:new()` and exposes the primary methods.
- JavaScript: `lib/gopherbot_v1.js` defines `new Robot()` and exposes the primary methods.

Both wrappers use the `GBOT` global injected by the interpreter modules (`lib/gopherbot_v1.lua`, `lib/gopherbot_v1.js`). They mirror most of the `robot.Robot` interface and are the canonical method list for Lua/JS extensions.

## External interpreter libraries (Bash / Python / Ruby)

External interpreters call the HTTP API and wrap it in language-appropriate helpers:

- Bash: `lib/gopherbot_v1.sh` exports functions like `Say`, `Reply`, `Remember`, `PromptForReply`, `AddTask`, and more; it uses curl to post JSON to `GOPHER_HTTP_POST`.
- Python 3: `lib/gopherbot_v2.py` defines `class Robot` with the same core methods, plus `Subscribe`, `Unsubscribe`, and `SetWorkingDirectory`.
- Ruby: `lib/gopherbot_v1.rb` defines `class Robot` (via `BaseBot`) with the same core methods, plus `Subscribe`, `Unsubscribe`, and `SetWorkingDirectory`.

## Parity notes and known gaps

- `Subscribe` / `Unsubscribe` exist in the engine (`bot/subscribe_thread.go`) and are exposed in the HTTP handler (`bot/http.go`), but they are not listed on the `robot.Robot` interface in `robot/robot.go`. TODO (verify): decide whether the Go interface should include these.
- `SetWorkingDirectory` exists in the Go interface and external libraries (`lib/gopherbot_v1.sh`, `lib/gopherbot_v2.py`, `lib/gopherbot_v1.rb`), but it is not present in the Lua/JS wrappers as of `lib/gopherbot_v1.lua` / `lib/gopherbot_v1.js`.
- `RaisePriv` is Go-only (`robot/robot.go`); there is no wrapper in external language libraries.

## Related docs

- `aidocs/INTERPRETERS.md` – execution model and interpreter categories
- `aidocs/EXTENSION_SURFACES.md` – where extensions live and how they are wired
