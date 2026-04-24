# SimpleMatcher DSL

`SimpleMatcher` provides a human-friendly syntax for matching user commands in Clu plugins. It is compiled into standard regular expressions under the hood. It allows developers to specify command formats naturally, without needing to write or escape complex regex strings.

**Note:** `SimpleMatcher` can only be used for directed commands (e.g. `bot, do something`), not for ambient message matching.

## Syntax Rules

| Syntax | Description | Example |
|---|---|---|
| **Literal Words** | Matches the word exactly, ignoring case. | `hello world` |
| **Hyphenated Words** | Hyphens are treated as flexible separators (matches spaces or hyphens). | `log-level` matches `log level`, `log-level`, `log  level` |
| **`<type>`** | An unnamed capture slot of a given type. Emits a regex capture group. | `<ident>` |
| **`<label:type>`** | A labeled capture slot. Emits a regex capture group. The `label:` is ignored by the compiler and is solely for human readability. | `<level:ident>` |
| **`[...]`** | A capturing optional group (if it contains no slots). If present, it creates a capture group containing the words. If omitted, it emits an empty capture group `""`. If the group contains slots, it behaves as a non-capturing optional to avoid double-captures. | `[disabled]` |
| **`{...}`** | A non-capturing noise-word group. Words inside are entirely optional and ignored (they do not create a capture group). | `{please}` |
| **`(a \| b)`** | Required alternation. Must match one of the choices. | `(start\|stop)` |
| **`{a \| b}`** | Optional alternation. May match one of the choices, or be omitted. | `{ticket\|story}` |

## Capture Types

When using slots like `<label:type>`, the `type` determines the regex pattern used. 

| Type | Matches | Regex | Example |
|---|---|---|---|
| `token` | Any non-whitespace | `[^\s]+` | `abc` |
| `ident` | Identifier | `[A-Za-z][\w-]*` | `slack-prod` |
| `number` | Integer | `[+-]?\d+` | `-42` |
| `decimal` | Float / Decimal | `[+-]?(?:\d+(?:\.\d+)?\|\.\d+)` | `3.14` |
| `bool` | Boolean | `(?:true\|false\|yes\|no\|on\|off\|1\|0)` | `true`, `on` |
| `url` | Full URL | `[A-Za-z][A-Za-z0-9+.-]*://[^\s]+` | `https://example.com` |
| `email` | Email address | `[^\s@]+@[^\s@]+\.[^\s@]+` | `ops@example.com` |
| `ip` / `ipv4` / `ipv6` | IP addresses | `(?:\d{1,3}\.){3}\d{1,3}` etc. | `10.0.0.1` |
| `duration` | Go-style durations | `(?:\d+(?:ns\|us\|ms\|s\|m\|h))+` | `5m30s` |
| `dnsname` | DNS hostname | standard DNS hostname regex | `api.example.com` |
| `slug` | Slug identifier | `[\w.*-]+` | `train-123*` |
| `rest` | Everything remaining | `.+` | `because prod is broken` |

## Understanding `<label:type>`

When you write a capture slot like `<level:ident>`, the word `level:` is just a **human-readable label**. 

Under the hood, `SimpleMatcher` compiles this down to a standard, *unnamed* capture group, like `([A-Za-z][\w-]*)`. 
The Go regex engine assigns it a standard numeric index (e.g. `matches[1]`). The `level:` label is completely ignored by the code — it exists purely so that developers reading the YAML configuration can understand what the `<ident>` represents.

You can write `<ident>` or `<level:ident>`, and both behave identically in the engine.

## Examples and Regex Mapping

Here is how `SimpleMatcher` strings are interpreted and mapped into regular expressions.
*(Note: All final regexes are wrapped in `^(?s:\s*(?i:PATTERN)\s*)$` to ensure full string matching, case-insensitivity, and padding safety. The examples below show the inner `PATTERN` for clarity.)*

### Example 1: Basic Command
**SimpleMatcher:** `ping`
**Generated Regex:** `ping`
**Matches:** `"ping"`, `"PING"`

### Example 2: Captures and Noise Words
**SimpleMatcher:** `set log-level {to} <level:ident>`
**Generated Regex:** `set(?:[ -]+)?log(?:[ -]+)?level(?:[ -]+to)?(?:[ -]+)?([A-Za-z][\w-]*)`
- `"set log level debug"` -> matches, group 1 is `"debug"`
- `"set log-level to info"` -> matches, group 1 is `"info"`
- `"set log level"` -> fails (missing `<ident>`)

### Example 3: Capturing Optionals
**SimpleMatcher:** `set log-level <level:ident> [disabled]`
**Generated Regex:** `set(?:[ -]+)?log(?:[ -]+)?level(?:[ -]+)?([A-Za-z][\w-]*)(?:[ -]+(disabled))?`
- `"set log-level debug disabled"` -> matches, group 1 is `"debug"`, group 2 is `"disabled"`
- `"set log-level debug"` -> matches, group 1 is `"debug"`, group 2 is `""`

**SimpleMatcher:** `ps [<mode:token>]`
**Generated Regex:** `ps(?:[ -]+([^\s]+))?`
- `"ps -v"` -> matches, group 1 is `"-v"`
- `"ps"` -> matches, group 1 is `""`

### Example 4: Alternations
**SimpleMatcher:** `(start|stop|restart) <service:ident>`
**Generated Regex:** `(?:start|stop|restart)(?:[ -]+)?([A-Za-z][\w-]*)`
- `"restart web"` -> matches, group 1 is `"web"`

## Important Restrictions

1. **Optional Groups and Slots**: You can put a capture slot `<...>` inside a capturing optional `[...]` (like `[<name:ident>]`). When you do this, the `[...]` wrapper automatically becomes non-capturing so you only get ONE capture group (the slot itself), preventing double-capture bugs. For pure literal optionals like `[disabled]`, it remains capturing.
2. **Cannot be empty**: The matcher must contain at least one literal, slot, or group.
3. **Cannot specify both**: A plugin configuration command cannot define both `Regex` and `SimpleMatcher`. You must choose one.
