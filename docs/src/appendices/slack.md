# A.1 Slack

The first and best-supported protocol is [Slack](https://slack.com). Developers wishing to support new protocols should consider Slack the "gold standard". **Gopherbot** uses the [slack-go/slack](https://github.com/slack-go/slack) library.

The `Message` struct for Slack will have an `.Protocol` value of `robot.Slack`, and `.Incoming` pointer to a `robot.ConnectorMessage` struct:

- `Protocol`: `slack`
- `MessageObject`: `*slack.MessageEvent`
- `Client`: `*slack.Client`

## Outgoing message formats

Slack is the most fully developed team-chat connector, but exact rendering can still vary a bit between Slack clients, notifications, and previews.

- `BasicMarkdown`
  - the normal v3 default for portable formatted output
- `Variable`
  - sent as literal variable-width text
- `Fixed`
  - sent as preformatted Slack content for code or aligned text
- `Raw`
  - passed through with connector-native interpretation

If you are trying to understand a formatting difference on Slack, start with the API formatting chapter and then use this appendix for Slack-specific expectations.
