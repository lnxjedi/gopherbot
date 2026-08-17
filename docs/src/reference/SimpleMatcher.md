# SimpleMatcher

`SimpleMatcher` is Gopherbot's recommended command-matching syntax for normal directed plugin commands. It gives plugin authors a command-line-like grammar without making them write regular expressions for every command. Use it when a command can be described as words, optional words, choices, typed values, and CLI-style dash options:

```yaml
Commands:
- Command: console
  SimpleMatcher: "get console [-options:-spot|-branch:<token>] [<environment:token>]"
  Usage: "get console [-spot] [-branch:<branch>] [environment]"
  Summary: "open an application console"
  Examples:
  - "(alias) get console qa"
  - "(alias) get console -branch:feature/login -spot qa"
```

For that matcher, a plugin invocation receives:

| User input | Plugin arguments |
|---|---|
| `get console qa` | `qa` |
| `get-console qa` | `qa` |
| `get console -spot qa` | `-spot`, `qa` |
| `get console -branch:feature/login -spot qa` | `-branch:feature/login`, `-spot`, `qa` |
| `get console` | `""` |

The first argument passed to a plugin is still the configured `Command` value. The values above are the command arguments after that command token.

## Where SimpleMatcher Is Used

`SimpleMatcher` is supported for directed plugin `Commands` in `conf/plugins/<plugin>.yaml`.

```yaml
Commands:
- Command: deploy
  SimpleMatcher: "deploy <service:ident> [<branch:token>]"
```

It is not used for:

- `MessageMatchers`
- `ReplyMatchers`
- job argument matchers
- connector-local command parsing

Those matcher types continue to use `Regex`.

## Why Use It

Prefer `SimpleMatcher` when the command should feel like a small command-line interface:

- it is easier to read in plugin YAML
- it makes argument order obvious
- it supports typed captures such as `<ident>`, `<number>`, and `<url>`
- it supports optional "noise" words without shifting arguments
- it can produce better user-facing diagnostics for invalid captured values
- it avoids regex escaping and grouping mistakes for common command shapes

Use `Regex` instead when you need arbitrary regular-expression behavior, unusual punctuation rules, lookarounds, complex repeated structures, or exact control over regex capture groups.

## Matching Basics

Simple matchers are case-insensitive. Leading and trailing whitespace around the command body is ignored, and extra whitespace between terms is accepted.

```yaml
SimpleMatcher: "service status <service:ident>"
```

Matches:

```text
service status api
SERVICE   STATUS   api
service-status api
```

Does not match:

```text
service_status api
service status
status service api
```

Directed command matching happens after the user has addressed the robot. If your bot alias is `;`, then `;service status api` is reduced to the command body `service status api` before this matcher is checked.

## Spaces And Dashes

Spaces between command words in the `SimpleMatcher` spec match either whitespace or dashes in user input.

```yaml
SimpleMatcher: "show build log"
```

Matches all of these:

```text
show build log
show-build-log
show   build-log
```

That dash forgiveness is for command wording only. Captured values must be separated from the preceding command words by real whitespace.

```yaml
SimpleMatcher: "rails up [<branch:token>]"
```

Matches:

```text
rails up
rails-up
rails up dev
rails-up dev
```

Does not match:

```text
rails-up-dev
```

This is deliberate. `rails-up-dev` should be available to match a separate command such as:

```yaml
SimpleMatcher: "rails up dev"
```

If you want CLI-like dash options, define an options block and require a space before each option:

```yaml
SimpleMatcher: "rails up [-options:-spot|-branch:<token>]"
```

Matches:

```text
rails up -spot
rails-up -branch:dev
```

Does not match:

```text
rails-up-spot
rails-up-branch:dev
```

## Complete Syntax

| Syntax | Meaning | Captures? |
|---|---|---:|
| `literal words` | Required command text. Spaces between command words can be typed as spaces or dashes. | No |
| `{word}` | Optional non-capturing text. Good for polite words or articles. | No |
| `{a\|b\|c}` | Optional non-capturing choices. | No |
| `/a\|b\|c/` | Required non-capturing synonyms. | No |
| `(label:a\|b\|c)` | Required capturing choice. | Yes |
| `(:a\|b\|c)` | Required capturing choice with no label. | Yes |
| `[label:a\|b\|c]` | Optional capturing choice. Adds `""` when omitted. | Yes |
| `[:a\|b\|c]` | Optional capturing choice with no label. Adds `""` when omitted. | Yes |
| `<type>` | Required typed capture. | Yes |
| `<name:type>` | Required typed capture with a diagnostic label. | Yes |
| `[<type>]` | Optional typed capture. Adds `""` when omitted. | Yes |
| `[<name:type>]` | Optional typed capture with a diagnostic label. Adds `""` when omitted. | Yes |
| `[-label:-flag\|-name:&lt;type&gt;]` | Optional options block. Matched options become individual arguments. Adds no argument when omitted. | Variable |

The punctuation characters are part of the SimpleMatcher grammar, not regex delimiters. If your command depends on regex escaping or regex-specific precedence, use `Regex`.

## Literal Words

Literal words are required command text.

```yaml
SimpleMatcher: "show log"
```

The matcher accepts `show log`, `show-log`, and `SHOW LOG`. Literal words are not passed as plugin arguments.

## Optional Noise Words

Use `{...}` for words that should be accepted but should not change plugin behavior.

```yaml
SimpleMatcher: "set log lines {to} <lines:number>"
```

Both inputs pass one argument, `3`:

```text
set log lines 3
set log lines to 3
```

Optional noise groups can include choices:

```yaml
SimpleMatcher: "{please|kindly} restart <service:ident>"
```

Use noise groups sparingly. They make commands more forgiving, but too many accepted phrasings can make help text harder to reason about.

## Required Synonyms

Use `/.../` when several required words mean the same thing to the plugin.

```yaml
SimpleMatcher: "/remove|delete/ <user:token> from {the} <group:rest> group"
```

Both inputs pass the same arguments, `alice` and `ops`:

```text
remove alice from ops group
delete alice from the ops group
```

Synonym groups can contain phrases:

```yaml
SimpleMatcher: "/pick up|take|grab/ <item:rest>"
```

This matches `pick up wrench`, `pick-up wrench`, `take wrench`, and `grab wrench`.

Do not use bare alternation such as `remove|delete <user:token>`. Bare `|` is intentionally not part of the top-level grammar.

## Capturing Choices

Use `(label:...)` when the selected value matters to the plugin.

```yaml
SimpleMatcher: "set log level {to} (level:trace|debug|info|warn|error)"
```

Inputs:

```text
set log level debug
set log-level to warn
```

Arguments:

```text
debug
warn
```

The label before the colon is used for diagnostics and documentation. The plugin receives only the selected value.

Capturing choices must include a label prefix. If you truly do not want a label, use an empty label:

```yaml
SimpleMatcher: "set target (:foo:bar|baz|frotz)"
```

That matcher accepts `foo:bar`, `baz`, or `frotz`. The first top-level colon separates the label from the choices, so the empty-label form is useful when choice values themselves contain colons.

Optional capturing choices use square brackets:

```yaml
SimpleMatcher: "set feature <name:ident> [state:disabled]"
```

Arguments:

| User input | Plugin arguments |
|---|---|
| `set feature cache disabled` | `cache`, `disabled` |
| `set feature cache` | `cache`, `""` |

## Typed Captures

Typed captures match user-supplied values and pass them to the plugin.

```yaml
SimpleMatcher: "deploy <service:ident> [<branch:token>]"
```

Arguments:

| User input | Plugin arguments |
|---|---|
| `deploy api main` | `api`, `main` |
| `deploy api` | `api`, `""` |

The name is optional:

```yaml
SimpleMatcher: "show <ident>"
SimpleMatcher: "show <service:ident>"
```

Both match the same values and pass the same argument. The named form is usually better for public commands because it gives the engine and the docs a human word for the value. For example, `<service:ident>` lets Gopherbot say that a bad value is invalid for `service`; `<ident>` falls back to the type name.

Use `[<name:type>]` for optional typed captures. When omitted, ordinary optional captures contribute an empty string argument so later argument positions stay stable.

```yaml
SimpleMatcher: "ps [<mode:token>]"
```

Arguments:

| User input | Plugin arguments |
|---|---|
| `ps -v` | `-v` |
| `ps` | `""` |

## Capture Types

| Type | Accepts | Examples |
|---|---|---|
| `token` | Any single non-whitespace token. | `main`, `feature/foo`, `-v` |
| `ident` | Identifier starting with a letter, followed by letters, numbers, `_`, or `-`. | `api`, `slack-prod` |
| `number` | Integer. | `0`, `42`, `-7` |
| `decimal` | Decimal number. | `3.14`, `.5`, `-2` |
| `bool` | Boolean-like value. | `true`, `false`, `yes`, `no`, `on`, `off`, `1`, `0` |
| `duration` | Go-style duration. | `30s`, `5m`, `1h30m` |
| `email` | Email address. | `ops@example.com` |
| `url` | Full URL with a scheme. | `https://example.com/runbook` |
| `ip` | IPv4 or IPv6 address. | `10.0.0.5`, `2001:db8::1` |
| `ipv4` | IPv4 address. | `10.0.0.5` |
| `ipv6` | IPv6 address. | `2001:db8::1` |
| `cidr` | IPv4 CIDR block. | `10.0.0.0/24` |
| `dnsname` | DNS hostname. | `api.example.com` |
| `slug` | Slug-like identifier containing word characters, `.`, `*`, or `-`. | `train-123`, `prod.*` |
| `base64` | Base64-looking text. | `QUJDRA==` |
| `rest` | The remaining non-empty text. | `because prod is broken` |

`rest` is greedy and should usually appear at the end of a matcher or optional group. It is useful for reasons, messages, descriptions, and other human text.

## Options Blocks

Options blocks are for CLI-like dash options that should be passed through as individual plugin arguments.

```yaml
SimpleMatcher: "get console [-options:-spot|-branch:<token>] [<environment:token>]"
```

The block:

- must be inside square brackets
- must have a label that starts with `-`, such as `-options`
- must contain one or more `|`-separated option forms
- requires every option form to start with `-`
- matches options only at the position where the block appears
- accepts options in any order
- allows repeated options
- passes each matched option exactly as the user typed it
- emits no argument when no options are present

Arguments:

| User input | Plugin arguments |
|---|---|
| `get console qa` | `qa` |
| `get console -spot qa` | `-spot`, `qa` |
| `get console -branch:feature/foo qa` | `-branch:feature/foo`, `qa` |
| `get console -spot -branch:feature/foo qa` | `-spot`, `-branch:feature/foo`, `qa` |
| `get console -spot -spot qa` | `-spot`, `-spot`, `qa` |
| `get console -spot` | `-spot`, `""` |
| `get console` | `""` |

Typed option values use a typed capture at the end of the option form:

```yaml
SimpleMatcher: "deploy [-options:-branch:<token>|-timeout:<duration>] <service:ident>"
```

Examples:

```text
deploy -branch:feature/login api
deploy -timeout:30s api
deploy -branch:main -timeout:1m api
```

The plugin receives `-branch:main` and `-timeout:1m` as whole strings. SimpleMatcher validates the typed part, but it does not split option names from option values. That parsing belongs in your plugin.

Options are positional in the command grammar:

```yaml
SimpleMatcher: "get console [-options:-spot|-branch:<token>] [<environment:token>]"
```

Matches:

```text
get console -spot qa
```

Does not match:

```text
get console qa -spot
```

If you use an options block before normal positional captures, argument positions become variable. A practical plugin pattern is:

1. Read leading arguments that start with `-` as options.
2. Stop option parsing at the first non-option argument.
3. Interpret the remaining arguments as positional values.

For example, `get console -branch:main -spot qa` yields:

```text
-branch:main
-spot
qa
```

Your plugin can consume the first two values as options and treat `qa` as the environment.

## Argument Ordering

Captured values are passed to the plugin in the order their capturing terms appear in the matcher, except that an options block can contribute any number of arguments at its position.

Non-capturing terms do not affect argument order:

- literal words
- optional noise groups `{...}`
- required synonym groups `/.../`

Ordinary optional captures add `""` when omitted:

If you want an optional phrase, put the whole phrase inside the optional group:

```yaml
SimpleMatcher: "copy <source:token> [to <destination:token>]"
```

Arguments:

| User input | Plugin arguments |
|---|---|
| `copy app to staging` | `app`, `staging` |
| `copy app` | `app`, `""` |

If `to` were outside the optional group, the word `to` would be required even when the destination was omitted.

An options block is different: when omitted, it adds no placeholder argument.

## Diagnostics

When a command has the right shape but a captured value is invalid, Gopherbot can return a specific syntax diagnostic instead of generic help.

```yaml
SimpleMatcher: "set loglevel {to} (level:trace|debug|info|warn|error)"
```

Input:

```text
set loglevel to fine
```

Possible reply:

```text
Invalid value: "fine" for: "level"; valid values: trace, debug, info, warn, error.
```

Typed captures can also report useful expectations:

```yaml
SimpleMatcher: "deploy siding <siding:ident>"
```

Input:

```text
deploy siding 9round
```

Possible reply:

```text
Invalid value: "9round" for: "siding"; expected: an identifier starting with a letter, followed by letters, numbers, '_' or '-'.
```

Diagnostics are intentionally conservative. If the command skeleton does not match exactly, the matcher returns no diagnostic and the normal help or fallback path runs.

For example:

```yaml
SimpleMatcher: "set loglevel {to} (level:trace|debug|info|warn|error)"
```

`set logging to fine` is not treated as a bad `level` value, because `logging` is not the same command skeleton as `loglevel`.

Exact command matches always win over syntax diagnostics. If more than one visible command could produce a diagnostic, Gopherbot avoids guessing and falls back to the normal unmatched-command behavior.

## Help Metadata

`SimpleMatcher` controls matching. It does not replace help metadata.

Always provide a user-facing `Usage`, `Summary`, `Examples`, and useful `Keywords`:

```yaml
Commands:
- Command: deploy
  SimpleMatcher: "deploy [-options:-branch:<token>|-wait:<bool>] <service:ident>"
  Usage: "deploy [-branch:<branch>] [-wait:<true|false>] <service>"
  Summary: "deploy a service"
  Examples:
  - "(alias) deploy api"
  - "(alias) deploy -branch:release/2026-07 api"
  Keywords: [ "deploy", "release", "ship" ]
```

Use `Usage` for the command users should type, not for the SimpleMatcher grammar itself. For example, `[-branch:<branch>]` is friendlier than `[-options:-branch:<token>]`.

## Common Patterns

### Simple command

```yaml
SimpleMatcher: "ping"
```

No arguments.

### Required typed value

```yaml
SimpleMatcher: "service status <service:ident>"
```

`service status api` passes `api`.

### Optional value

```yaml
SimpleMatcher: "show logs [page <page:number>]"
```

`show logs page 2` passes `2`; `show logs` passes `""`.

### Required choice

```yaml
SimpleMatcher: "set log level (level:trace|debug|info|warn|error)"
```

Passes the selected level.

### Synonyms

```yaml
SimpleMatcher: "/remove|delete/ <user:token>"
```

Accepts either verb and passes only the user.

### Noise word

```yaml
SimpleMatcher: "show {the} <group:rest> group"
```

Accepts `show ops group` and `show the ops group`; passes `ops`.

### CLI-like options

```yaml
SimpleMatcher: "rails up [-options:-spot|-branch:<token>] [<environment:token>]"
```

Accepts `rails up -spot qa` and `rails-up -branch:feature/foo qa`; passes the options as separate leading arguments, followed by the environment.

## Choosing The Right Form

Use `(label:...)` when the selected required choice changes plugin behavior.

Use `[label:...]` or `[<name:type>]` when an optional value should occupy a stable argument position and become `""` when omitted.

Use `[-label:...]` when you want CLI-like dash options that can appear in any order and should become argv-style arguments.

Use `{...}` for words that are accepted only to make the command more natural.

Use `/.../` for required synonyms whose selected spelling should not matter to the plugin.

Use `<name:type>` for values supplied by the user.

## Restrictions And Gotchas

A command matcher must specify exactly one of `Regex` or `SimpleMatcher`.

A `SimpleMatcher` cannot be empty.

Capturing choices must include a colon:

```yaml
# Good
SimpleMatcher: "set level (level:trace|debug)"
SimpleMatcher: "set level (:trace|debug)"

# Invalid
SimpleMatcher: "set level (trace|debug)"
```

Non-capturing groups cannot contain typed captures:

```yaml
# Invalid
SimpleMatcher: "show {<name:ident>}"
SimpleMatcher: "/show <name:ident>|list/"
```

Capturing choice groups cannot contain typed captures:

```yaml
# Invalid
SimpleMatcher: "show (target:<name:ident>|all)"
```

Use an optional group with a typed capture instead:

```yaml
SimpleMatcher: "show [<name:ident>]"
```

Options must be single tokens and must start with `-`:

```yaml
# Good
SimpleMatcher: "deploy [-options:-spot|-branch:<token>] <service:ident>"

# Invalid
SimpleMatcher: "deploy [-options:spot|-branch:<token>] <service:ident>"
SimpleMatcher: "deploy [-options:-branch:<token>-extra] <service:ident>"
```

Spaces before captures are real boundaries. If you have both `rails up [<branch:token>]` and `rails up dev`, then `rails-up-dev` belongs to `rails up dev`, not to the optional branch capture on `rails up`.

## Migration From Regex

A regex command like:

```yaml
Commands:
- Command: deploy
  Regex: '(?i:deploy ([A-Za-z][A-Za-z0-9_-]*)(?: ([^\\s]+))?)'
```

can usually become:

```yaml
Commands:
- Command: deploy
  SimpleMatcher: "deploy <service:ident> [<branch:token>]"
```

Check these items when migrating:

- Match the same command body, not the robot alias or bot name.
- Replace non-capturing regex alternatives with `/.../`.
- Replace meaningful alternatives with `(label:...)`.
- Choose a typed capture that fits the value.
- Remember that optional captures emit `""` when omitted.
- Add an options block only when options must start with `-` and should become argv-style arguments.
- Keep `Usage`, `Summary`, `Examples`, and `Keywords` friendly for users.

Regex remains the right choice for uncommon shapes. SimpleMatcher is intended to make normal command authoring boring, readable, and predictable.
