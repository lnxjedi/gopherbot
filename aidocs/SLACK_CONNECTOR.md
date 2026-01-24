# Slack Connector Notes

This file captures recent dependency-driven API shifts for the Slack connector.

## slack-go v0.17.x changes observed

- `api.GetBotInfo` now requires `slack.GetBotInfoParameters` (bot + team ID).
- `slackevents.MessageEvent` no longer exposes attachments/timestamps directly;
  use the embedded `Message` payload (`*slack.Msg`) for `Attachments`, `Timestamp`,
  and `ThreadTimestamp`.
