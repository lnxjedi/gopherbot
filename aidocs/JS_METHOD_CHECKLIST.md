# JavaScript Extension Method Checklist

Purpose: track JS parity with the canonical Robot API (`robot/robot.go`) and define JS add-on helpers needed for DevOps-oriented extensions. Check items off once verified in the JS runtime and library (`lib/gopherbot_v1.js`).

Sources:
- Canonical API: `robot/robot.go` (type `Robot`)
- External API names: `bot/http.go` (handler `ServeHTTP`)
- JS library: `lib/gopherbot_v1.js`

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
- [ ] `RaisePriv(path)` (if JS should expose it)

### Thread subscription (engine support)
- [x] `Subscribe()`
- [x] `Unsubscribe()`

## JS add-on helpers (DevOps focus)

### HTTP / remote APIs
- [x] `gopherbot_http.createClient(...).request(options)` – method, URL/path, headers, body, timeout; returns status, headers, body.
- [x] `client.getJSON(path, options)` / `client.postJSON(path, payload, options)` – JSON helpers.

### Local file access (workspace-safe)
- [ ] `ReadFile(path, options)` – respects workspace/current working dir.
- [ ] `WriteFile(path, data, options)` – safe writes; create dirs if requested.
- [ ] `ListDir(path, options)`
- [ ] `Stat(path)`

### Local command execution
- [ ] `Exec(command, args, options)` – timeout, cwd, env overrides, stdin; returns exit code, stdout, stderr.

## JS full test coverage targets

Track coverage for the JS full test extension under `test/` once it exists.

- Deferred beyond this epic:
  - `SetWorkingDirectory(path)` (track as long-term; not in current JS/Lua full coverage slices)

- [ ] Messaging: Say/Reply/Send* variants + format wrappers
- [ ] Prompting: all Prompt* variants including thread/user/channel
- [ ] Memory: Checkout/Update/Recall + Remember* context
- [ ] Pipeline control: Add/Final/Fail task/job/command, SpawnJob
- [ ] Admin + Elevate
- [x] Subscribe/Unsubscribe
- [ ] File helpers
- [ ] Exec helpers (success + failure + timeout)
- [ ] HTTP helpers (status, JSON parse error, timeout)
