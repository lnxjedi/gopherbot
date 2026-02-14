# Lua Extension Method Checklist

Purpose: track Lua parity with the canonical Robot API (`robot/robot.go`) and define Lua add-on helpers needed for DevOps-oriented extensions. Check items off once verified in the Lua runtime and library (`lib/gopherbot_v1.lua`).

Sources:
- Canonical API: `robot/robot.go` (type `Robot`)
- External API names: `bot/http.go` (handler `ServeHTTP`)
- Lua library: `lib/gopherbot_v1.lua`

## Core Robot API (parity checklist)

### Identity, attributes, config
- [ ] `GetMessage()`
- [ ] `GetTaskConfig()`
- [ ] `GetParameter(name)`
- [ ] `GetBotAttribute(attr)`
- [ ] `GetUserAttribute(user, attr)`
- [ ] `GetSenderAttribute(attr)`

### Messaging and formatting
- [ ] `Direct()`
- [ ] `Threaded()`
- [ ] `Fixed()`
- [ ] `MessageFormat(format)`
- [ ] `SendChannelMessage(channel, message, format?)`
- [ ] `SendChannelThreadMessage(channel, thread, message, format?)`
- [ ] `SendUserChannelMessage(user, channel, message, format?)`
- [ ] `SendProtocolUserChannelMessage(protocol, user, channel, message, format?)`
- [ ] `SendUserChannelThreadMessage(user, channel, thread, message, format?)`
- [ ] `SendUserMessage(user, message, format?)`
- [ ] `Say(message, format?)`
- [ ] `SayThread(message, format?)`
- [ ] `Reply(message, format?)`
- [ ] `ReplyThread(message, format?)`

### Prompting
- [ ] `PromptForReply(regexId, prompt, format?)`
- [ ] `PromptThreadForReply(regexId, prompt, format?)`
- [ ] `PromptUserForReply(regexId, user, prompt, format?)`
- [ ] `PromptUserChannelForReply(regexId, user, channel, prompt, format?)`
- [ ] `PromptUserChannelThreadForReply(regexId, user, channel, thread, prompt, format?)`

### Memory (brain + ephemeral)
- [ ] `CheckoutDatum(key, rw)`
- [ ] `CheckinDatum(memory)`
- [ ] `UpdateDatum(memory)`
- [ ] `Remember(key, value, shared)`
- [ ] `RememberThread(key, value, shared)`
- [ ] `RememberContext(context, value)`
- [ ] `RememberContextThread(context, value)`
- [ ] `Recall(key, shared)`

### Pipeline control
- [ ] `Exclusive(tag, queueTask)`
- [ ] `SpawnJob(name, ...args)`
- [ ] `AddTask(name, ...args)`
- [ ] `FinalTask(name, ...args)`
- [ ] `FailTask(name, ...args)`
- [ ] `AddJob(name, ...args)`
- [ ] `AddCommand(pluginName, command)`
- [ ] `FinalCommand(pluginName, command)`
- [ ] `FailCommand(pluginName, command)`

### Admin, logging, utilities
- [ ] `CheckAdmin()`
- [ ] `Elevate(immediate)`
- [ ] `Log(level, message)`
- [ ] `RandomInt(n)`
- [ ] `RandomString(list)`
- [ ] `Pause(seconds)`
- [ ] `Email(...)` / `EmailUser(...)` / `EmailAddress(...)`

### Workspace + privilege
- [ ] `SetParameter(name, value)`
- [ ] `SetWorkingDirectory(path)`
- [ ] `RaisePriv(path)` (if Lua should expose it)

### Thread subscription (engine support)
- [x] `Subscribe()`
- [x] `Unsubscribe()`

## Lua add-on helpers (DevOps focus)

### HTTP / remote APIs
- [x] `gopherbot_http.create_client(...):request(options)` – method, URL/path, headers, body, timeout; returns status, headers, body.
- [x] `client:get_json(path, options)` / `client:post_json(path, payload, options)` / `client:put_json(path, payload, options)` – JSON helpers.

### Local file access (workspace-safe)
- [ ] `ReadFile(path, options)` – respects workspace/current working dir.
- [ ] `WriteFile(path, data, options)` – safe writes; create dirs if requested.
- [ ] `ListDir(path, options)`
- [ ] `Stat(path)`

### Local command execution
- [ ] `Exec(command, args, options)` – timeout, cwd, env overrides, stdin; returns exit code, stdout, stderr.

## Lua full test coverage targets

Track coverage for the Lua full test extension under `test/`.

- Deferred beyond this epic:
  - `SetWorkingDirectory(path)` (track as long-term; not in current JS/Lua full coverage slices)

- [x] Messaging: Say/Reply/Send* variants + format wrappers
- [x] Config: GetTaskConfig + RandomString
- [x] HTTP helpers (status, error, timeout)
- [ ] Prompting: all Prompt* variants including thread/user/channel
- [ ] Memory: Checkout/Update/Recall + Remember* context
- [ ] Pipeline control: Add/Final/Fail task/job/command, SpawnJob
- [ ] Admin + Elevate
- [x] Subscribe/Unsubscribe
- [ ] File helpers
- [ ] Exec helpers (success + failure + timeout)
