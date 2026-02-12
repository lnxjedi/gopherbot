# Slack Connector Notes

This file captures recent dependency-driven API shifts for the Slack connector.

## slack-go v0.17.x changes observed

- `api.GetBotInfo` now requires `slack.GetBotInfoParameters` (bot + team ID).
- `slackevents.MessageEvent` no longer exposes attachments/timestamps directly;
  use the embedded `Message` payload (`*slack.Msg`) for `Attachments`, `Timestamp`,
  and `ThreadTimestamp`.

## Runtime Lifecycle Notes

- Slack connector runtime state is connector-instance scoped (not package-global).
- Outbound queueing and edited-message dedupe tracking are maintained per connector instance.
- This supports in-process lifecycle operations used by multi-protocol runtime management:
  - `protocol-stop slack`
  - `protocol-start slack`
  - `protocol-restart slack`
  - remove/add `slack` in `SecondaryProtocols` across reloads
