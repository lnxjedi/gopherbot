# A.1 Slack

The first and best-supported protocol is [Slack](https://slack.com). Developers wishing to support new protocols should consider Slack the "gold standard". **Gopherbot** uses the [slack-go/slack](https://github.com/slack-go/slack) library.

The `Message` struct for Slack will have an `.Protocol` value of `robot.Slack`, and `.Incoming` pointer to a `robot.ConnectorMessage` struct:

* `Protocol`: "slack"
* `MessageObject`: `*slack.MessageEvent`
* `Client`: `*slack.Client`