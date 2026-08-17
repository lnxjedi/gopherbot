# BasicMarkdown Reference

`BasicMarkdown` is the default outgoing message format in Gopherbot v3. It is a small, portability-first formatting language intended to let one plugin produce readable output across Slack, SSH, and other connectors without relying on connector-specific markup.

For send methods and format-selection helpers such as `MessageFormat(...)`, `Fixed()`, and `Direct()`, see [Message Sending and Formatting](api/Message-Sending-API.md).

## Design goals

- keep shared automation portable
- support the formatting operators most teams actually need
- degrade cleanly on simpler connectors
- avoid visibly broken connector-specific syntax

## Supported syntax

### Paragraphs and line breaks

A blank line starts a new paragraph. A single newline is a line break.

```text
Hello team,

The deploy has started.
Investigating now.
```

### Bold

Use double asterisks.

```text
**important**
```

### Italic

Use single asterisks.

```text
*emphasis*
```

### Inline code

Use backticks.

```text
Use `kubectl get pods` to inspect the namespace.
```

### Fenced code blocks

Use triple backticks. An optional language hint is allowed.

````text
```yaml
apiVersion: v1
kind: Pod
```
````

### Block quotes

Use `>` at the start of the line.

```text
> deploy paused pending approval
```

### Unordered lists

Use `- ` at the start of a line.

```text
- build
- deploy
- verify
```

Only a single list level is part of the current contract.

### Links

Use standard Markdown link syntax.

```text
[runbook](https://example.com/runbook)
```

If a connector cannot render labeled links natively, it should degrade to readable text such as `runbook (https://example.com/runbook)`.

### Mentions

`BasicMarkdown` supports connector-resolved mention tokens:

```text
@alice please confirm the production deploy.
```

Connectors should turn that into a native mention when they can resolve the user safely. If they cannot, the literal `@alice` text should remain visible.

Mentions are not parsed inside inline code or fenced code blocks.

### Emoji

Both shortcode emoji and raw Unicode emoji are allowed:

```text
:white_check_mark: deploy complete
```

```text
Deploy complete ✅
```

Connectors may render shortcode emoji natively when supported. If a shortcode is unknown, the literal text should remain visible.

## Escaping

Use backslash escapes when you want literal formatting characters in `BasicMarkdown` text:

```text
\*not italic\*
\`not code\`
\@literal-user
\[literal link text\]
```

## Connector expectations

Connectors are responsible for translating `BasicMarkdown` into native rendering on the target platform.

The contract is:

- preserve meaning first
- prefer faithful rendering where possible
- degrade to readable plain text where necessary
- never emit obviously broken formatting

For connector-specific rendering notes, check the connector appendices such as [Slack](appendices/slack.md).
