# Interpreter Decisions

Gopherbot supports two execution families:

- `.lua`, `.js`, `.gsh`, and interpreted `.go` run in one-shot
  `pipeline-child-rpc` processes. Robot API calls return to the parent over
  versioned stdio RPC.
- Other executable scripts (Bash, Python, Ruby, etc.) run as child processes and
  call the parent's localhost JSON API.

The parent remains the authority for policy, identity, brain access, parameter
scope, logging, cancellation, and outbound routing. Do not move those concerns
into language modules.

## Language contract

- Plugin handlers receive the engine command first, followed by matcher
  captures. `_configure` returns/prints YAML; lifecycle callbacks use the
  engine-reserved leading-underscore namespace.
- Built-in runtimes do not require standalone language installations. Use
  `gopherbot syntax` and `gopherbot script` for local work.
- Gopherbot shell intentionally provides Robot methods and common utilities as
  builtins so shipped automation can avoid host shell/jq dependencies.
- External scripts retain the HTTP library compatibility surface.

## Environment and filesystem

`HOME` and `PATH` retain launcher/user meaning. `GOPHER_HOME` is the robot root.
`Homed` and `SetWorkingDirectory` affect only the child/pipeline execution
context, never the parent process working directory.

Lua and JavaScript built-in runtimes deliberately do not expose ambient
`GOPHER_*` through `os.getenv`; use `GetParameter`, which resolves context via
the parent. GSH exposes its engine-constructed shell environment. External
executables receive the constructed environment for compatibility.

Ruby dependencies live under `$GOPHER_HOME/.bot-gems`; Python user packages
under `$GOPHER_HOME/.bot-python`. These paths must remain usable by
unprivileged child UIDs.

## Interpreted Go

Installed shared libraries import as `gopherbot.internal/lib/...`; custom robot
libraries import as `robot.internal/lib/...`. The engine-managed Yaegi GOPATH
links these roots.

Yaegi reflection can reject multi-value helper returns that compiled Go accepts
(`reflect.Set ... not assignable`). When values cross interpreted helper
boundaries, prefer a single wrapper struct. Keep the focused Yaegi repro tests
before "simplifying" such wrappers.

## HTTP modules

Lua and JavaScript expose synchronous `require("http")` modules. They buffer
complete responses; HTTP error status is data, not a transport exception.
Transport/timeout failures follow each language's ordinary error convention.
See `JS_HTTP_API.md` and `LUA_HTTP_API.md` for the small compatibility notes.
