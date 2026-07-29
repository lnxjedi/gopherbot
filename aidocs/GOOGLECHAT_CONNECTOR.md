# Google Chat Connector Decisions

Google Chat uses the Chat API outbound and Pub/Sub inbound. When ambient
messages are enabled, the connector also owns per-space Workspace Events
subscription creation, renewal, suspension recovery, and deletion. Terraform
owns project resources; it does not own these short-lived per-space resources.

Do not region-pin the shared Pub/Sub topic with a narrow message-storage policy.
Google may publish Workspace Events from another region, causing apparently
healthy subscriptions to become suspended later.

Outbound create retries reuse the same request ID to avoid duplicate messages
after ambiguous timeouts. Inbound interaction and ambient deliveries share a
seen-message cache because Google can deliver the same message through both
paths.

## Identity and context

- `UserMap` is connector-local canonical username → `users/{id}` mapping and is
  the only basis for `ValidatedUser=true`.
- `SelfID` is separate recognition metadata for the bot and never user policy.
  `HearSelf` remains engine-owned.
- Ordinary mentions are normalized and remain ordinary messages. Slash/app
  commands are bot-directed; configured slash commands are hidden/private.
- Hidden replies remain viewer-private only while the original user and space
  context are preserved. Retargeting drops hidden treatment.
- Direct messages have no engine channel identity.

Only `UserMap` reloads live and must swap atomically. Credentials, Pub/Sub,
ambient behavior, and self identity require restart.

## Transport choices

Google Chat may default same-context replies into the incoming thread. This is
connector-local and explicit thread APIs still win.

The shared surface is text-only: no cards/dialogs. `BasicMarkdown` is translated
to Chat text syntax. `Fixed` and `Variable` are visual approximations because
Chat lacks a general literal-text escape; do not promise byte-for-byte
copy/paste fidelity. `Raw` is non-portable and still subject to Chat parsing.
