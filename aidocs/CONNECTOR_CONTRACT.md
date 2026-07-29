# Connector Contract

Connectors translate transports; they do not implement shared policy.

## Inbound trust boundary

- Normalize transport markup into readable text while preserving meaning.
- Ordinary mentions remain ordinary text. Set `BotMessage=true` only when the
  transport itself routed an interaction unambiguously to this bot.
- Set `HiddenMessage=true` only for a transport-private invocation.
- `DirectMessage`, `HiddenMessage`, and `SelfMessage` are connector-authoritative.
  Forward recognized self messages; the engine's `HearSelf` policy decides
  whether to process them.
- A connector may provide a readable username for display, but
  `ValidatedUser=true` means it can prove that the transport account maps to
  that canonical username. Never set it for guesses, display-name matches, or
  heuristics.

Canonical username is the only cross-protocol policy identity. Transport IDs
remain connector-local routing data.

## Runtime behavior

- Connector initialization reports errors to the engine. The engine decides
  fatality: primary is required; secondaries are isolated and retryable.
- Reload only applies connector-local mutable state. Parse and normalize first,
  then swap atomically so readers see one complete version.
- Reconnect-level settings remain restart concerns.
- Preserve ordering within a connector; do not promise global order across
  protocols.
- Connector-specific default threading is allowed only as a transport behavior;
  it must not redefine engine send semantics.
- Native hidden-command connectors should provide the formatter used by
  engine-owned help and denial text.

## Privacy

Private-command eligibility, channel restrictions, authorization, and elevation
stay in the engine. A connector may preserve the invoking user for a hidden
reply, but must not broaden private visibility when target user/context changes.
