# Gopherbot Local Script Examples

These examples are runnable with `gopherbot script` and checkable with
`gopherbot syntax`. They are intentionally small but broad: each language shows
plugin commands, `_configure`, message APIs, config, prompting, memory, and
pipeline helpers.

Build the local binary first:

```sh
make
```

Syntax-check all examples:

```sh
./gopherbot syntax test-scripts/lua/demo.lua test-scripts/javascript/demo.js test-scripts/gsh/demo.gsh test-scripts/go/demo.go
```

Run plugin commands with a fixture:

```sh
./gopherbot script -fixture test-scripts/fixtures/cat.json test-scripts/lua/demo.lua -- demo from-cli
./gopherbot script -fixture test-scripts/fixtures/cat.json test-scripts/javascript/demo.js -- prompt
./gopherbot script -fixture test-scripts/fixtures/cat.json test-scripts/gsh/demo.gsh -- memory
./gopherbot script -fixture test-scripts/fixtures/cat.json test-scripts/go/demo.go -- config
```

Use JSON output when another tool or AI context should inspect the run:

```sh
./gopherbot script -json -fixture test-scripts/fixtures/cat.json test-scripts/lua/demo.lua -- prompt
```

Prompt behavior is deterministic when the fixture has `prompts.replies`. When
the replies are exhausted, `script` reads one line from stdin by default. Add
`-no-interactive` to turn an exhausted prompt into `TimeoutExpired`.

The path is explicit: `test-scripts/lua/demo.lua` is resolved from your current
working directory. The command after `--` is the plugin command; remaining
values are capture args.
