# Component Ownership

This map records boundaries that are easy to blur. Use `rg`, Go package
navigation, and tests for the current file/symbol inventory.

- `bot/` is the engine and policy authority: startup, configuration, routing,
  pipelines, authorization/elevation, brain/cache orchestration, and runtime
  lifecycle.
- `robot/` is the shared contract package. It must not depend on `bot/`.
- `connectors/` owns transport normalization, delivery, protocol-local IDs,
  and accurate DM/hidden/self context. It does not own business policy.
- `brains/`, `history/`, and `queues/` are provider implementations behind
  engine-owned contracts. Provider failure must not create an alternate policy
  path.
- `modules/` hosts built-in interpreter/runtime support. File-backed extension
  calls cross a child-process boundary; policy remains in `bot/`.
- `goplugins/`, `gojobs/`, and `gotasks/` are compiled into the engine and are
  trusted in-process code.
- `plugins/`, `jobs/`, and `tasks/` are file-backed extensions.
- `conf/` is the installed baseline. `robot.skel/` is the minimal custom-robot
  scaffold. A deployed robot's `custom/conf/` should contain deltas, not copied
  defaults.
- `integration/` and `cmd/gopherbot-integration/` own process-backed suites.
  `test/` contains the robot configurations and extension fixtures those
  suites run against, plus terminal-connector test support.
- `cmd/gopherbot-mcp/` provides lifecycle and compact integration automation.
- `devdocs/` documents extension authoring; `aidocs/` documents architecture
  decisions; root `UPGRADING-v3.md` owns migration instructions.

Cross-cutting changes should converge on engine paths rather than adding
connector/provider-specific exceptions.
