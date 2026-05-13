`robot/` is the shared contract surface between the Gopherbot engine and modular components.

It defines the types and registration APIs that components should depend on instead of importing engine internals from `bot/`.

Current shared surfaces here include:
- robot runtime/message contracts (`robot/robot.go`, `robot/connector_defs.go`)
- connector registrations and capabilities (`robot/connectors.go`)
- brain-provider registrations (`robot/brains.go`)
- history-provider registrations (`robot/history_providers.go`)
- queue-provider registrations and queue message contracts (`robot/queues.go`)
- compiled Go plugin/job/task registrations (`robot/registrations.go`)
- pure connector-safe helpers in `robot/util/` (`wrap.go`, `id.go`, `basic_markdown_plain.go`)

Design rule:
- modular components like connectors, brains, and history providers should import `github.com/lnxjedi/gopherbot/robot`
- they should not need visibility into `bot/` internals

Usage:
```go
import "github.com/lnxjedi/gopherbot/robot"
```
