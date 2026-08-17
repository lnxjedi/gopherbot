# Slack Connector Decisions

Slack maps configured canonical usernames to Slack IDs in connector-local
`UserMap`. Only that mapping may set `ValidatedUser=true`; a Slack display name
is readable metadata, not security identity.

Ordinary mentions are normalized to readable `@username` text and remain
`BotMessage=false` so engine addressing stays consistent across protocols.
Configured slash commands are the exception: Slack already routed them to this
app, so they are hidden and bot-directed.

Hidden-command capability is decided during initialization because it depends
on explicit slash-command configuration. A malformed enabled slash surface is
an initialization error; the engine decides fatality by primary/secondary role.
Help and denial examples use the configured slash command, not the bot display
name.

`UserMap` is the only live-reload surface. Build the replacement indexes before
an atomic swap; connection/client settings require restart.

## Formatting intent

- `Raw` remains Slack-specific compatibility behavior.
- `Variable` and `Fixed` use Block Kit rich-text/preformatted blocks so the
  visible body is not reparsed and remains copyable.
- `BasicMarkdown` uses Slack's native markdown-text path, resolving configured
  mentions while leaving code spans/blocks literal.
- Application chunk limits are deliberately conservative; preserve fallback
  text for notifications/accessibility.

Exact formatting and SDK adaptation belong in connector tests and source, not
this decision record.

## Outbound delivery

Slack outbound calls are completion-coupled: `Ok` means every chunk produced
for that Robot API call was accepted by Slack's Web API. Queue rejection,
connector shutdown, deadline expiry, or final API failure returns
`FailedMessageSend`. A failure after an earlier chunk succeeded is still a
whole-call failure; callers that retry must tolerate a duplicated prefix.

The connector keeps a single outbound worker to preserve message order and
burst control. Transient transport, rate-limit, and server failures receive
bounded retries; permanent Slack API errors do not. Delivery has an overall
deadline so an unhealthy Slack service cannot block a pipeline indefinitely.
There is no RTM fallback because RTM cannot confirm acceptance and would make a
reported failure ambiguous.

Do not add cosmetic pre-send latency. A successful call returns as soon as
Slack accepts its final chunk; burst throttling may delay later queued sends.
