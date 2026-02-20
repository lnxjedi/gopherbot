# BasicMarkdown Message Format Specification (BasicMarkdown v1)

## Purpose

BasicMarkdown is a **portable, minimal formatting format** for Gopherbot outgoing messages to be available in Gopherbot v3. Gopherbot v2 supported "Fixed", "Variable", and "Raw", defaulting to "Raw", which was expected to be recognized and specially formatted by the specific protocol - which led to a lot of Slack-specific formatting. With the introduction of BasicMarkdown, Gopherbot will use this as the default going forward (which older robots can override to "Raw"), but robot owners should look to update custom extensions to use BasicMarkdown.

It is designed to:

- Let plugins and core bot logic produce one message format
- Let each connector render that format for its target platform
- Avoid connector-specific formatting syntax in shared code
- Degrade safely when a platform cannot render a feature exactly

BasicMarkdown is intentionally a **small subset of Markdown**, with one Gopherbot extension:

- `@username` is interpreted as a reference to a given username from the UserRoster, and if supported by the platform should "mention" the user (some platforms support user notifications if a user is mentioned).

This spec is the **contract** that all connectors must follow for messages sent in `BasicMarkdown` mode.

---

## Normative language

The key words **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT**, and **MAY** are to be interpreted as described in RFC 2119.

---

## Design principles

BasicMarkdown is **portability-first**, not “full Markdown”.

Connectors MUST prefer:

1. **Faithful rendering** on the target platform
2. **Clean degradation** to readable plain text
3. **Never** emitting visibly broken formatting syntax

If a platform does not support a feature, the connector MUST degrade it to plain text in a way that preserves meaning.

---

## Supported features (v1)

BasicMarkdown v1 supports only the following constructs:

### 1) Paragraphs and line breaks

Plain text is allowed.

A blank line separates paragraphs.

A single newline is a line break.

#### Example

Hello team,

Prometheus is crashing again.
Investigating now.

---

### 2) Bold

Use double asterisks:

`**bold text**`

#### Requirements

- Connectors MUST render bold if supported.
- If unsupported, connectors MUST emit the text without markers.

---

### 3) Italic

Use single asterisks:

`*italic text*`

#### Requirements

- Connectors MUST render italic if supported.
- If unsupported, connectors MUST emit the text without markers.

---

### 4) Inline code

Use backticks:

`` `code` ``

#### Requirements

- Connectors MUST preserve inline code semantics if supported.
- If unsupported, connectors MUST emit the inner text as plain text (without backticks) or a safe equivalent.
- Connectors SHOULD preserve exact characters inside inline code as much as possible.

---

### 5) Fenced code blocks

Use triple backticks:

````
```text
multi-line code
or log output
```
````

An optional language hint MAY be supplied after the opening fence, but connectors are not required to use it:

````
```yaml
apiVersion: v1
kind: Pod
```
````

#### Requirements

* Connectors MUST render code blocks if supported.
* If unsupported, connectors MUST degrade to fixed-width/plain text with clear separation from surrounding text.
* Connectors MUST preserve line breaks and indentation inside code blocks.
* Connectors MUST NOT parse formatting inside code blocks.

---

### 6) Block quotes

Use `>` at the start of the line:

`> quoted text`

Multiple quoted lines are allowed.

#### Requirements

* Connectors MUST render quote style if supported.
* If unsupported, connectors MUST preserve the `>` prefix as plain text.

---

### 7) Unordered lists (single level)

Use `- ` (dash + space) at the start of a line:

`- item`

Only a **single list level** is part of v1.

#### Requirements

* Connectors MUST render unordered lists if supported.
* If unsupported, connectors MUST preserve the `- ` prefixes as plain text.
* Connectors MUST NOT assume nested list support.

---

### 8) Links

Use standard Markdown link syntax:

`[label](https://example.com/path)`

#### Requirements

* Connectors MUST render links if supported.
* If a platform does not support labeled links, connectors MUST degrade to:

  * `label (https://example.com/path)`
* Connectors SHOULD leave bare URLs clickable if the platform auto-linkifies them.

#### Security / safety note

Connectors SHOULD avoid emitting platform-specific link syntax that obscures the destination URL if the connector cannot preserve the original URL exactly.

---

### 9) Mentions (Gopherbot extension)

BasicMarkdown defines a Gopherbot-specific mention token:

`@username`

This is **not** standard Markdown. It is a connector-resolved semantic token.

#### Intent

The connector SHOULD convert `@username` into a real platform mention (notification + clickable mention) when supported and when the user can be resolved unambiguously.

#### Requirements

* Connectors MUST attempt to resolve `@username` to a platform-native mention when the platform supports mentions.
* If resolution fails (unknown user, ambiguous user, connector lacks lookup capability), connectors MUST leave the text as literal `@username`.
* Connectors MUST NOT silently mention the wrong user.
* Connectors SHOULD support connector-specific username mapping (e.g., aliases, IDs, canonical usernames).
* Connectors MAY require exact-case or normalized lookup internally, but user-visible behavior SHOULD be case-insensitive where practical.

#### Parsing rules for mentions

A mention token is recognized only when:

* `@` is followed by one or more of: `A-Z`, `a-z`, `0-9`, `_`, `-`, `.`
* It is not inside inline code or fenced code blocks
* It is not part of an email address (e.g. `name@example.com` is NOT a mention)

#### Examples

* `@david` → mention token
* `hello@server` → not a mention
* `` `@david` `` → not a mention (inline code)
* Code block content → not parsed for mentions

---

## Unsupported features (v1)

The following are intentionally out of scope for BasicMarkdown v1:

* Headings (`#`, `##`, etc.)
* Ordered lists
* Nested lists
* Tables
* Images
* HTML
* Horizontal rules
* Strikethrough
* Underline
* Task lists / checkboxes
* Footnotes
* Raw platform formatting syntax (Slack mrkdwn, HTML tags, etc.)

Connectors MUST treat unsupported syntax as plain text unless future versions of this spec define it.

---

## Escaping rules

BasicMarkdown v1 supports escaping with backslash (`\`) for literal formatting characters.

The following characters MAY be escaped when literal text is desired:

* `\*`
* ``\` ``
* `\[`
* `\]`
* `\(`
* `\)`
* `\@`
* `\\`

#### Requirements

* Connectors MUST process escapes before rendering platform-specific output.
* Connectors MUST preserve the literal character (without the backslash) when escaped.
* Escapes inside fenced code blocks MUST NOT be processed.

---

## Parsing and rendering order

Connectors (or shared formatting code) SHOULD parse BasicMarkdown in this order:

1. Fenced code blocks
2. Inline code
3. Links
4. Mentions
5. Bold / italic
6. Block quotes / lists / paragraphs

This order avoids accidental formatting inside code and reduces ambiguity.

---

## Connector conformance requirements

A connector that claims BasicMarkdown support MUST:

* Parse and render all v1 supported features
* Respect the mention token semantics (`@username`)
* Degrade unsupported target-platform features safely
* Avoid leaking raw, broken syntax into user-visible messages
* Preserve message readability under degradation

A connector SHOULD:

* Use platform-native formatting/mention constructs where available
* Escape target-platform reserved characters correctly
* Preserve user intent when exact formatting is not possible

A connector MAY:

* Implement additional platform-specific enhancements internally
* Support richer rendering than BasicMarkdown expresses
* Expose a separate `Raw` mode for platform-native syntax (outside this spec)

---

## Degradation examples (normative)

### Example A: labeled link on a platform without labeled-link support

Input:

`See [runbook](https://example.com/runbook).`

Required degraded output:

`See runbook (https://example.com/runbook).`

---

### Example B: quote on a platform without quote support

Input:

`> node is NotReady`

Required degraded output:

`> node is NotReady`

---

### Example C: unresolved mention

Input:

`Paging @david for review`

If `@david` cannot be resolved unambiguously, required output:

`Paging @david for review`

(Connector MUST NOT mention a guessed user.)

---

## Compatibility target (informative)

BasicMarkdown v1 is intentionally chosen to be implementable on common team chat platforms used by engineering teams, including Slack, Mattermost, Rocket.Chat, Google Chat, and Zulip, while also allowing clean SSH/plain-text rendering.

Because platform syntax differs, connectors are responsible for translation. BasicMarkdown is the stable contract.

---

## Versioning

This document defines **BasicMarkdown v1**.

Future versions SHOULD be additive and preserve backward compatibility where practical.

If a future version introduces new syntax, connectors that only support v1 MUST treat unknown syntax as plain text.

---

## Recommended test cases (informative)

Connector implementations should be tested with at least:

* Plain paragraphs
* Bold + italic in same message
* Inline code with `*` characters inside
* Fenced code blocks with indentation
* Quote lines
* Simple bullet lists
* Labeled links and bare URLs
* Mentions:

  * resolvable
  * unknown
  * ambiguous
  * inside code
  * email-address false positive
* Escaped formatting characters
* Mixed content (list + code + link + mention)

---

## Example message

````basicmarkdown
**Deploy status:** failed

> prod-us-east-1 rollout paused

- Service: payments-api
- Namespace: prod
- Owner: @david

Logs:
```text
CrashLoopBackOff: back-off 5m0s restarting failed container
````

See [runbook](https://example.com/runbooks/payments-api).

```

Expected connector behavior:

- Render formatting where supported
- Resolve `@david` to a platform mention if possible
- Preserve code block formatting exactly
- Degrade safely on limited platforms

```
