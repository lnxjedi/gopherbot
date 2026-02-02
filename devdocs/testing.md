# Testing

## Quick commands

- `make test` runs unit tests (`go test ./...`) and the standard integration suite.
- `make fulltest` runs unit tests plus the full integration suite (JS + Lua).
- `make testbot` builds the test bot binary used by integration tests.

## What runs where

- Unit tests: `go test ./...` (fast checks for logic and invariants).
- Integration tests: `go test -v --tags 'test integration netgo osusergo static_build' -mod readonly -race ./test`.
- Full integration tests: same as integration, but with `RUN_FULL=all`.

## Nudge: more unit tests welcome

Integration tests provide good coverage for end-to-end behavior, but we still have gaps
in unit tests around connector and routing logic. If you’re touching behavior that routes
or transforms messages, consider adding a small unit test to guard it.
