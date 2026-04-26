# SimpleMatcher DSL

`SimpleMatcher` is the v3 command-matcher DSL for Gopherbot plugins. It is intended to cover normal chat command shapes without forcing plugin authors to write Go regular expressions. The engine compiles it to regex during config load, and matched capture groups are passed positionally to the plugin handler.

Status note: this document is the v3 authoring contract. The parser and shipped `SimpleMatcher` configs are expected to follow this bracket model; `bot/simple_matcher_test.go` includes representative shipped-pattern argument-position coverage.

`SimpleMatcher` is only for directed plugin `Commands`, such as `bot, do something` or alias-prefixed hidden commands. Ambient `MessageMatchers`, reply matchers, and job argument matchers remain regex-based.

## Design Rule

Use `SimpleMatcher` for the common 99% of command matchers. If a command needs escaping rules, lookarounds, unusual punctuation handling, or subtle regex precedence, use `Regex` instead.

## Syntax Rules

| Syntax | Meaning | Captures? | Example |
|---|---|---:|---|
| `literal words` | Required literal text. Spaces in the spec match spaces or dashes in input. | No | `show log` matches `show log` and `show-log` |
| `/a|b/` | Required non-capturing synonyms. Use when choices are equivalent to the plugin. | No | `/remove|delete/ <user:token>` |
| `(a|b)` | Required capturing choice. Use when the plugin needs the selected value. | Yes | `(trace|debug|info|warn|error)` |
| `[a|b]` | Optional capturing choice or phrase. If omitted, the argument is `""`. | Yes | `[disabled]` |
| `{a|b}` | Optional non-capturing noise text. Use only to widen accepted wording. | No | `{please|kindly}` |
| `<type>` | Required typed capture slot. | Yes | `<ident>` |
| `<label:type>` | Required typed capture slot with a human-readable label. | Yes | `<level:ident>` |

The delimiter characters are part of the SimpleMatcher grammar, not regex syntax. Do not rely on regex escaping inside a `SimpleMatcher`; switch to `Regex` when the simple grammar is not enough.

## Capture Semantics

Captured values are passed to plugin handlers in the order their capturing syntax appears. Literal text, required synonym groups (`/.../`), and optional noise groups (`{...}`) do not shift argument indexes.

Examples:

```yaml
SimpleMatcher: "set log level {to} (trace|debug|info|warn|error)"
```

```text
set log level debug     -> args[0] = "debug"
set log level to debug  -> args[0] = "debug"
```

```yaml
SimpleMatcher: "/remove|delete/ <user:token> from {the} <group:rest> group"
```

```text
remove alice from ops group      -> args[0] = "alice", args[1] = "ops"
delete alice from the ops group  -> args[0] = "alice", args[1] = "ops"
```

```yaml
SimpleMatcher: "set feature <name:ident> [disabled]"
```

```text
set feature cache disabled -> args[0] = "cache", args[1] = "disabled"
set feature cache          -> args[0] = "cache", args[1] = ""
```

When a bracketed group contains a typed capture slot, the slot is the semantic capture. The wrapper should not add a second capture group around the slot.

```yaml
SimpleMatcher: "ps [<mode:token>]"
```

```text
ps -v -> args[0] = "-v"
ps    -> args[0] = ""
```

Use non-capturing noise for words that should not affect plugin behavior:

```yaml
SimpleMatcher: "show {the} <group:rest> group"
```

`the` is accepted but never passed to the plugin.

## Required Synonyms

Use `/.../` for required synonyms where the selected word should not matter to the handler:

```yaml
SimpleMatcher: "/start|launch/ <service:ident>"
```

This avoids ambiguous bare `foo|bar` precedence and keeps `(...)` available for choices that are meaningful plugin input.

Phrases are allowed inside synonym groups:

```yaml
SimpleMatcher: "/pick up|take|grab/ <item:rest>"
```

Do not use bare `foo|bar` in a SimpleMatcher. It is intentionally not part of the grammar.

## Capture Types

When using slots like `<label:type>`, the `type` determines the regex pattern used.

| Type | Matches | Example |
|---|---|---|
| `token` | Any non-whitespace token | `abc`, `foo/bar` |
| `ident` | Identifier starting with a letter | `slack-prod` |
| `number` | Integer | `-42` |
| `decimal` | Decimal number | `3.14` |
| `bool` | Boolean-ish values | `true`, `on`, `0` |
| `url` | Full URL | `https://example.com` |
| `email` | Email address | `ops@example.com` |
| `ip` / `ipv4` / `ipv6` | IP addresses | `10.0.0.1` |
| `duration` | Go-style duration | `5m30s` |
| `dnsname` | DNS hostname | `api.example.com` |
| `slug` | Slug-like identifier | `train-123*` |
| `rest` | Everything remaining | `because prod is broken` |

The label in `<label:type>` is documentation only. `<level:ident>` and `<ident>` compile to the same capture behavior; the label helps humans understand what the captured value represents.

## Choosing The Right Brackets

- Use `(...)` when the selected required choice changes what the plugin should do.
- Use `[...]` when the optional value itself is meaningful to the plugin.
- Use `{...}` when optional words only make the command easier to say.
- Use `/.../` when required words are synonyms and should not become plugin arguments.
- Use `<label:type>` for typed values supplied by the user.

## Restrictions

1. A matcher cannot specify both `Regex` and `SimpleMatcher`.
2. A `SimpleMatcher` cannot be empty.
3. `SimpleMatcher` does not support arbitrary regex syntax. Use `Regex` for the uncommon cases that need it.
